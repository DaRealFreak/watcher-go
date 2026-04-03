package schalenetwork

import (
	"context"
	"fmt"
	"log/slog"
	"path"
	"time"

	watcherHttp "github.com/DaRealFreak/watcher-go/internal/http"
	"github.com/DaRealFreak/watcher-go/internal/http/tls_session"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
)

type proxySession struct {
	inUse         bool
	proxy         watcherHttp.ProxySettings
	session       *tls_session.TlsClientSession
	occurredError error
}

func (m *schaleNetwork) initializeProxySessions() {
	m.proxies = make([]*proxySession, 0)

	mainSession, ok := m.Session.(*tls_session.TlsClientSession)
	if !ok {
		slog.Error("cannot share cookie jar: main session is not a TlsClientSession", "module", m.Key)
		return
	}

	sharedJar := mainSession.Jar

	for _, proxy := range m.settings.LoopProxies {
		if !proxy.Enable {
			continue
		}

		singleSession := tls_session.NewTlsClientSessionWithJar(m.Key, sharedJar)

		// use Firefox 147 profile for each proxy session
		client, _ := tls_client.NewHttpClient(tls_client.NewNoopLogger(),
			tls_client.WithTimeoutSeconds(30*60),
			tls_client.WithClientProfile(profiles.Firefox_147),
			tls_client.WithRandomTLSExtensionOrder(),
			tls_client.WithCookieJar(sharedJar),
		)
		singleSession.SetClient(client)

		raven.CheckError(singleSession.SetProxy(&proxy))
		m.proxies = append(m.proxies, &proxySession{
			inUse:         false,
			proxy:         proxy,
			session:       singleSession,
			occurredError: nil,
		})
	}
}

func (m *schaleNetwork) isLowestIndex(index int) bool {
	lowestIndex := 999
	for _, v := range m.multiProxy.currentIndexes {
		if v < lowestIndex {
			lowestIndex = v
		}
	}

	return lowestIndex == index
}

func (m *schaleNetwork) hasFreeProxy() bool {
	for _, proxy := range m.proxies {
		if !proxy.inUse {
			return true
		}
	}

	return false
}

func (m *schaleNetwork) getFreeProxy() *proxySession {
	for _, proxy := range m.proxies {
		if !proxy.inUse {
			return proxy
		}
	}

	return nil
}

func (m *schaleNetwork) getProxyError() *proxySession {
	for _, proxy := range m.proxies {
		if proxy.occurredError != nil {
			return proxy
		}
	}

	return nil
}

func (m *schaleNetwork) resetProxies() {
	for _, proxy := range m.proxies {
		proxy.inUse = false
		proxy.occurredError = nil
	}
}

func (m *schaleNetwork) processDownloadQueueMultiProxy(
	downloadQueue []models.DownloadQueueItem,
	trackedItem *models.TrackedItem,
	notifications ...*models.Notification,
) error {
	slog.Info(
		fmt.Sprintf("found %d new items for uri: \"%s\"", len(downloadQueue), trackedItem.URI),
		"module", m.Key,
	)

	for _, notification := range notifications {
		slog.Log(context.Background(), notification.Level, notification.Message, "module", m.Key)
	}

	for index, data := range downloadQueue {
		// sleep until we have a free proxy again
		for !m.hasFreeProxy() && m.getProxyError() == nil {
			time.Sleep(time.Millisecond * 100)
		}

		// handle if errors occurred in previous downloads
		if erroneousProxy := m.getProxyError(); erroneousProxy != nil {
			slog.Warn(fmt.Sprintf("error occurred during download for proxy: %s",
				erroneousProxy.proxy.Host), "module", m.Key)
			return m.getProxyError().occurredError
		}

		if m.hasFreeProxy() {
			slog.Info(fmt.Sprintf(
				"downloading updates for uri: \"%s\" (%0.2f%%)",
				trackedItem.URI,
				float64(index+1)/float64(len(downloadQueue))*100,
			), "module", m.Key)

			m.multiProxy.waitGroup.Add(1)
			proxy := m.getFreeProxy()
			proxy.inUse = true

			m.multiProxy.currentIndexes = append(m.multiProxy.currentIndexes, index)

			go m.downloadItemSession(proxy, trackedItem, data, index)

			time.Sleep(time.Millisecond * 100)
		}
	}

	m.multiProxy.waitGroup.Wait()

	// handle if errors occurred in previous downloads
	if erroneousProxy := m.getProxyError(); erroneousProxy != nil {
		slog.Warn(fmt.Sprintf("error occurred during download for proxy: %s",
			erroneousProxy.proxy.Host), "module", m.Key)
		return m.getProxyError().occurredError
	}

	// if no error occurred, update the tracked item to the last item ID
	if len(downloadQueue) > 0 {
		m.DbIO.UpdateTrackedItem(trackedItem, downloadQueue[len(downloadQueue)-1].ItemID)
	}

	return nil
}

func (m *schaleNetwork) downloadItemSession(
	downloadSession *proxySession,
	trackedItem *models.TrackedItem,
	data models.DownloadQueueItem,
	index int,
) {
	filePath := path.Join(
		m.GetDownloadDirectory(),
		m.Key,
		fp.TruncateMaxLength(fp.SanitizePath(trackedItem.SubFolder, false)),
		fp.TruncateMaxLength(fp.SanitizePath(data.DownloadTag, false)),
		fp.TruncateMaxLength(fp.SanitizePath(data.FileName, false)),
	)

	downloadSession.occurredError = m.downloadFileWithSession(downloadSession.session, filePath, data.FileURI)

	if downloadSession.occurredError == nil {
		if m.isLowestIndex(index) {
			m.DbIO.UpdateTrackedItem(trackedItem, data.ItemID)
		}

		for i, v := range m.multiProxy.currentIndexes {
			if v == index {
				m.multiProxy.currentIndexes = append(m.multiProxy.currentIndexes[:i], m.multiProxy.currentIndexes[i+1:]...)
				break
			}
		}
	} else {
		slog.Error(fmt.Sprintf("error occurred downloading item %s (%s) with proxy %s: %s",
			trackedItem.URI, data.ItemID, downloadSession.proxy.Host, downloadSession.occurredError.Error()), "module", m.Key)
	}

	downloadSession.inUse = false
	m.multiProxy.waitGroup.Done()
}

// downloadFileWithSession downloads a file using the given session with proper headers
func (m *schaleNetwork) downloadFileWithSession(session *tls_session.TlsClientSession, filePath string, uri string) error {
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", "https://niyaniya.moe/")
	req.Header.Set("Origin", "https://niyaniya.moe")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "cross-site")

	if m.settings.Cloudflare.UserAgent != "" {
		req.Header.Set("User-Agent", m.settings.Cloudflare.UserAgent)
	}

	resp, err := session.Do(req)
	if err != nil {
		return err
	}

	return session.DownloadFileFromResponse(resp, filePath)
}
