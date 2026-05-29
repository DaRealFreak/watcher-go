package momonga

import (
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"log/slog"

	"github.com/DaRealFreak/watcher-go/internal/http"
	"github.com/DaRealFreak/watcher-go/internal/http/tls_session"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
	"golang.org/x/time/rate"
)

type proxySession struct {
	inUse         bool
	proxy         http.ProxySettings
	session       *tls_session.TlsClientSession
	occurredError error
}

// initializeProxySessions builds one session per enabled loop proxy, sharing the main cookie jar
func (m *momonga) initializeProxySessions() {
	m.proxies = make([]*proxySession, 0)

	mainSession, ok := m.Session.(*tls_session.TlsClientSession)
	if !ok {
		slog.Error("cannot share cookie jar: main session is not a TlsClientSession")
		return
	}
	sharedJar := mainSession.Jar

	for _, proxy := range m.settings.LoopProxies {
		if !proxy.Enable {
			continue
		}

		singleSession := tls_session.NewTlsClientSessionWithJar(m.Key, sharedJar)
		singleSession.RateLimiter = rate.NewLimiter(rate.Every(time.Duration(m.rateLimit)*time.Millisecond), 1)
		raven.CheckError(singleSession.SetProxy(&proxy))
		m.proxies = append(m.proxies, &proxySession{
			proxy:   proxy,
			session: singleSession,
		})
	}
}

func (m *momonga) isLowestIndex(index int) bool {
	m.multiProxy.mutex.Lock()
	defer m.multiProxy.mutex.Unlock()

	lowestIndex := 999
	for _, v := range m.multiProxy.currentIndexes {
		if v < lowestIndex {
			lowestIndex = v
		}
	}

	return lowestIndex == index
}

func (m *momonga) hasFreeProxy() bool {
	m.multiProxy.mutex.Lock()
	defer m.multiProxy.mutex.Unlock()

	for _, proxy := range m.proxies {
		if !proxy.inUse {
			return true
		}
	}

	return false
}

// acquireFreeProxy atomically reserves a free proxy session, or returns nil if none is available
func (m *momonga) acquireFreeProxy() *proxySession {
	m.multiProxy.mutex.Lock()
	defer m.multiProxy.mutex.Unlock()

	for _, proxy := range m.proxies {
		if !proxy.inUse {
			proxy.inUse = true
			return proxy
		}
	}

	return nil
}

func (m *momonga) getProxyError() *proxySession {
	m.multiProxy.mutex.Lock()
	defer m.multiProxy.mutex.Unlock()

	for _, proxy := range m.proxies {
		if proxy.occurredError != nil {
			return proxy
		}
	}

	return nil
}

func (m *momonga) resetProxies() {
	m.multiProxy.mutex.Lock()
	defer m.multiProxy.mutex.Unlock()

	for _, proxy := range m.proxies {
		proxy.inUse = false
		proxy.occurredError = nil
	}
	m.multiProxy.currentIndexes = nil
}

func (m *momonga) removeCurrentIndex(index int) {
	m.multiProxy.mutex.Lock()
	defer m.multiProxy.mutex.Unlock()

	for i, v := range m.multiProxy.currentIndexes {
		if v == index {
			m.multiProxy.currentIndexes = append(m.multiProxy.currentIndexes[:i], m.multiProxy.currentIndexes[i+1:]...)
			break
		}
	}
}

func (m *momonga) processDownloadQueueMultiProxy(downloadQueue []models.DownloadQueueItem, trackedItem *models.TrackedItem) error {
	slog.Info(fmt.Sprintf("found %d new items for uri: \"%s\"", len(downloadQueue), trackedItem.URI), "module", m.Key)

	for index, data := range downloadQueue {
		// wait until a proxy frees up or a previous download errored
		for !m.hasFreeProxy() && m.getProxyError() == nil {
			time.Sleep(time.Millisecond * 100)
		}

		if erroneousProxy := m.getProxyError(); erroneousProxy != nil {
			slog.Warn(fmt.Sprintf("error occurred during download for proxy: %s",
				erroneousProxy.proxy.Host), "module", m.Key)
			m.multiProxy.waitGroup.Wait()
			return erroneousProxy.occurredError
		}

		proxy := m.acquireFreeProxy()
		if proxy == nil {
			// unreachable in the single-producer model: the wait loop above only exits
			// with a free proxy to claim or a proxy error (handled above). guard defensively.
			continue
		}

		slog.Info(fmt.Sprintf(
			"downloading updates for uri: \"%s\" (%0.2f%%)",
			trackedItem.URI,
			float64(index+1)/float64(len(downloadQueue))*100,
		), "module", m.Key)

		m.multiProxy.waitGroup.Add(1)

		m.multiProxy.mutex.Lock()
		m.multiProxy.currentIndexes = append(m.multiProxy.currentIndexes, index)
		m.multiProxy.mutex.Unlock()

		go m.downloadItemSession(proxy, trackedItem, data, index)
	}

	m.multiProxy.waitGroup.Wait()

	if erroneousProxy := m.getProxyError(); erroneousProxy != nil {
		slog.Warn(fmt.Sprintf("error occurred during download for proxy: %s",
			erroneousProxy.proxy.Host), "module", m.Key)
		return erroneousProxy.occurredError
	}

	// if no error occurred, update the tracked item to the last item ID
	if len(downloadQueue) > 0 {
		m.DbIO.UpdateTrackedItem(trackedItem, downloadQueue[len(downloadQueue)-1].ItemID)
	}

	return nil
}

func (m *momonga) downloadItemSession(
	downloadSession *proxySession, trackedItem *models.TrackedItem, downloadQueueItem models.DownloadQueueItem, index int,
) {
	defer m.multiProxy.waitGroup.Done()

	if err := m.downloadImageSession(downloadSession, trackedItem, downloadQueueItem, index); err != nil {
		m.multiProxy.mutex.Lock()
		downloadSession.occurredError = err
		m.multiProxy.mutex.Unlock()
	}

	m.multiProxy.mutex.Lock()
	downloadSession.inUse = false
	m.multiProxy.mutex.Unlock()
}

func (m *momonga) downloadImageSession(
	downloadSession *proxySession, trackedItem *models.TrackedItem, downloadQueueItem models.DownloadQueueItem, index int,
) error {
	startTime := time.Now()

	// apply rate limit for the current session since we don't use the wrapper function
	downloadSession.session.ApplyRateLimit()

	res, err := downloadSession.session.GetClient().Get(downloadQueueItem.FileURI)
	if err != nil {
		return err
	}

	if res.StatusCode == 429 {
		slog.Warn(fmt.Sprintf("received status code 429 on image url \"%s\"",
			downloadQueueItem.FileURI), "module", m.Key)
		time.Sleep(time.Second * 5)

		return m.downloadImageSession(downloadSession, trackedItem, downloadQueueItem, index)
	}

	if res.StatusCode == 404 {
		// the image just doesn't exist (anymore); log and skip rather than fail the gallery
		slog.Warn(fmt.Sprintf("received status code 404 on image url \"%s\"",
			downloadQueueItem.FileURI), "module", m.Key)
		return nil
	}

	dst := path.Join(
		m.GetDownloadDirectory(),
		m.Key,
		fp.TruncateMaxLength(fp.SanitizePath(strings.TrimSpace(trackedItem.SubFolder), false)),
		fp.TruncateMaxLength(strings.TrimSpace(downloadQueueItem.DownloadTag)),
		fp.TruncateMaxLength(strings.TrimSpace(downloadQueueItem.FileName)),
	)

	if downloadErr := downloadSession.session.DownloadFileFromResponse(res, dst); downloadErr != nil {
		return downloadErr
	}

	// bump the file's mtime so that ordering by time == ordering by page index
	if info, statErr := os.Stat(dst); statErr == nil && !info.IsDir() {
		if chtErr := os.Chtimes(dst, startTime, startTime); chtErr != nil {
			slog.Warn(fmt.Sprintf("failed to reset timestamp for %s: %v", dst, chtErr), "module", m.Key)
		}
	}

	if m.isLowestIndex(index) {
		// only the lowest in-flight index advances progress, to prevent skips on errors
		m.DbIO.UpdateTrackedItem(trackedItem, downloadQueueItem.ItemID)
	}

	m.removeCurrentIndex(index)

	return nil
}
