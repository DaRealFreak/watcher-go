package ehentai

import (
	"fmt"
	"github.com/DaRealFreak/watcher-go/internal/http"
	"github.com/DaRealFreak/watcher-go/internal/http/std_session"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
	log "github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
	"net/url"
	"path"
	"strings"
	"time"
)

type proxySession struct {
	inUse         bool
	proxy         http.ProxySettings
	session       *std_session.StdClientSession
	occurredError error
}

func (m *ehentai) initializeProxySessions() {
	// reset the multi-proxy sessions
	m.proxies = make([]*proxySession, 0)

	// copy the cookies for e-hentai to exhentai
	ehURL, _ := url.Parse("https://e-hentai.org")
	exURL, _ := url.Parse("https://exhentai.org")

	for _, proxy := range m.settings.LoopProxies {
		singleSession := std_session.NewStdClientSession(m.Key, ErrorHandler{}, std_session.StdClientErrorHandler{})
		singleSession.RateLimiter = rate.NewLimiter(rate.Every(time.Duration(m.rateLimit)*time.Millisecond), 1)
		// copy login cookies for session
		singleSession.SetCookies(ehURL, m.Session.GetCookies(ehURL))
		singleSession.SetCookies(exURL, m.Session.GetCookies(ehURL))
		raven.CheckError(singleSession.SetProxy(&proxy))
		m.proxies = append(m.proxies, &proxySession{
			inUse:         false,
			proxy:         proxy,
			session:       singleSession,
			occurredError: nil,
		})
	}
}

func (m *ehentai) isLowestIndex(index int) bool {
	lowestIndex := 999
	for _, v := range m.multiProxy.currentIndexes {
		if v < lowestIndex {
			lowestIndex = v
		}
	}

	return lowestIndex == index
}

func (m *ehentai) hasFreeProxy() bool {
	for _, proxy := range m.proxies {
		if !proxy.inUse {
			return true
		}
	}

	return false
}

func (m *ehentai) getFreeProxy() *proxySession {
	for _, proxy := range m.proxies {
		if !proxy.inUse {
			return proxy
		}
	}

	return nil
}

func (m *ehentai) getProxyError() *proxySession {
	for _, proxy := range m.proxies {
		if proxy.occurredError != nil {
			return proxy
		}
	}

	return nil
}

func (m *ehentai) resetProxies() {
	for _, proxy := range m.proxies {
		proxy.inUse = false
		proxy.occurredError = nil
	}
}

func (m *ehentai) processDownloadQueueMultiProxy(downloadQueue []*imageGalleryItem, trackedItem *models.TrackedItem, notifications ...*models.Notification) error {
	log.WithField("module", m.Key).Info(
		fmt.Sprintf("found %d new items for uri: \"%s\"", len(downloadQueue), trackedItem.URI),
	)

	for _, notification := range notifications {
		log.WithField("module", m.Key).Log(
			notification.Level,
			notification.Message,
		)
	}

	for index, data := range downloadQueue {
		// sleep until we have a free proxy again
		for !m.hasFreeProxy() && m.getProxyError() == nil {
			time.Sleep(time.Millisecond * 100)
			continue
		}

		// handle if errors occurred in previous downloads
		if erroneousProxy := m.getProxyError(); erroneousProxy != nil {
			log.WithField("module", m.Key).Warnf(
				"error occurred during download for proxy: %s",
				erroneousProxy.proxy.Host,
			)
			return m.getProxyError().occurredError
		}

		if m.hasFreeProxy() {
			log.WithField("module", m.Key).Info(
				fmt.Sprintf(
					"downloading updates for uri: \"%s\" (%0.2f%%)",
					trackedItem.URI,
					float64(index+1)/float64(len(downloadQueue))*100,
				),
			)

			m.multiProxy.waitGroup.Add(1)
			proxy := m.getFreeProxy()
			proxy.inUse = true

			m.multiProxy.currentIndexes = append(m.multiProxy.currentIndexes, index)

			go m.downloadItemSession(proxy, trackedItem, data, index)
		}

	}

	m.multiProxy.waitGroup.Wait()

	// handle if errors occurred in previous downloads
	if erroneousProxy := m.getProxyError(); erroneousProxy != nil {
		log.WithField("module", m.Key).Warnf(
			"error occurred during download for proxy: %s",
			erroneousProxy.proxy.Host,
		)
		return m.getProxyError().occurredError
	}

	// if no error occurred, update the tracked item to the last item ID
	if len(downloadQueue) > 0 {
		m.DbIO.UpdateTrackedItem(trackedItem, downloadQueue[len(downloadQueue)-1].id)
	}

	return nil
}

func (m *ehentai) downloadItemSession(
	downloadSession *proxySession, trackedItem *models.TrackedItem, data *imageGalleryItem, index int,
) {
	downloadQueueItem, err := m.getDownloadQueueItem(downloadSession.session, trackedItem, data)
	if err != nil {
		downloadSession.occurredError = err
		downloadSession.inUse = false
		m.multiProxy.waitGroup.Done()
		return
	}

	if err = m.downloadImageSession(downloadSession, trackedItem, downloadQueueItem, index); err != nil {
		if downloadQueueItem.FallbackFileURI == "" {
			downloadSession.occurredError = err
			downloadSession.inUse = false
			m.multiProxy.waitGroup.Done()
			return
		}

		// we have a fallback URI
		data.uri = downloadQueueItem.FallbackFileURI
		fallback, fallbackErr := m.getDownloadQueueItem(downloadSession.session, trackedItem, data)
		if fallbackErr != nil {
			downloadSession.occurredError = fallbackErr
			downloadSession.inUse = false
			m.multiProxy.waitGroup.Done()
			return
		}

		log.WithField("module", m.Key).Warnf(
			"received status code 404 on gallery url \"%s\", trying fallback url \"%s\"",
			data.uri,
			fallback.FileURI,
		)

		downloadQueueItem.FileURI = fallback.FileURI
		downloadQueueItem.FallbackFileURI = ""

		// retry the fallback once and override the previous error with the new result
		downloadSession.occurredError = m.downloadImageSession(downloadSession, trackedItem, downloadQueueItem, index)
	}

	downloadSession.inUse = false
	m.multiProxy.waitGroup.Done()
}

func (m *ehentai) downloadImageSession(
	downloadSession *proxySession, trackedItem *models.TrackedItem, downloadQueueItem *models.DownloadQueueItem, index int,
) error {
	// check for limit
	if downloadQueueItem.FileURI == "https://exhentai.org/img/509.gif" ||
		downloadQueueItem.FileURI == "https://e-hentai.org/img/509.gif" {
		log.WithField("module", m.Key).Info("download limit reached, skipping galleries from now on")
		m.downloadLimitReached = true

		return fmt.Errorf("download limit reached")
	}

	downloadErr := downloadSession.session.DownloadFile(
		path.Join(
			m.GetDownloadDirectory(),
			m.Key,
			fp.TruncateMaxLength(fp.SanitizePath(strings.TrimSpace(trackedItem.SubFolder), false)),
			fp.TruncateMaxLength(strings.TrimSpace(downloadQueueItem.DownloadTag)),
			fp.TruncateMaxLength(strings.TrimSpace(downloadQueueItem.FileName)),
		),
		downloadQueueItem.FileURI,
	)

	if downloadErr == nil {
		downloadSession.inUse = false

		if m.isLowestIndex(index) {
			// if we are the lowest index (to prevent skips on errors) update the downloaded item
			m.DbIO.UpdateTrackedItem(trackedItem, downloadQueueItem.ItemID)
		}

		// remove current index from current list since we finished
		for i, v := range m.multiProxy.currentIndexes {
			if v == index {
				m.multiProxy.currentIndexes = append(m.multiProxy.currentIndexes[:i], m.multiProxy.currentIndexes[i+1:]...)
				break
			}
		}
	}

	return downloadErr
}
