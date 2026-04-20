package ehentai

import (
	"fmt"
	"path"
	"strings"
	"time"

	"context"
	"github.com/DaRealFreak/watcher-go/internal/http"
	"github.com/DaRealFreak/watcher-go/internal/http/std_session"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
	"golang.org/x/time/rate"
	"log/slog"
)

type proxySession struct {
	inUse         bool
	limitReached  bool
	proxy         http.ProxySettings
	session       *std_session.StdClientSession
	occurredError error
}

func (m *ehentai) initializeProxySessions() {
	// reset the multi-proxy sessions
	m.proxies = make([]*proxySession, 0)

	// share the cookie jar from the main session so all proxy sessions stay in sync
	mainSession, ok := m.Session.(*std_session.StdClientSession)
	if !ok {
		slog.Error("cannot share cookie jar: main session is not a StdClientSession")
		return
	}

	sharedJar := mainSession.Jar

	for _, proxy := range m.settings.LoopProxies {
		if !proxy.Enable {
			continue
		}

		singleSession := std_session.NewStdClientSessionWithJar(m.Key, sharedJar, ErrorHandler{}, std_session.StdClientErrorHandler{})
		singleSession.RateLimiter = rate.NewLimiter(rate.Every(time.Duration(m.rateLimit)*time.Millisecond), 1)
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

func (m *ehentai) hasFreeProxy() bool {
	for _, proxy := range m.proxies {
		if !proxy.inUse && !proxy.limitReached {
			return true
		}
	}

	return false
}

// acquireFreeProxy atomically reserves a free, non-limit-reached proxy session.
// Returns nil if none is currently available.
func (m *ehentai) acquireFreeProxy() *proxySession {
	m.multiProxy.mutex.Lock()
	defer m.multiProxy.mutex.Unlock()

	for _, proxy := range m.proxies {
		if !proxy.inUse && !proxy.limitReached {
			proxy.inUse = true
			return proxy
		}
	}

	return nil
}

// waitForFreeProxy blocks until a usable proxy becomes available or all proxies have hit their
// individual download limits. Returns nil in the latter case.
func (m *ehentai) waitForFreeProxy() *proxySession {
	for {
		if m.allProxiesLimitReached() {
			return nil
		}

		if proxy := m.acquireFreeProxy(); proxy != nil {
			return proxy
		}

		time.Sleep(time.Millisecond * 100)
	}
}

func (m *ehentai) allProxiesLimitReached() bool {
	for _, proxy := range m.proxies {
		if !proxy.limitReached {
			return false
		}
	}

	return true
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
		// limitReached persists across galleries - a proxy that has hit its daily quota
		// stays excluded from rotation for the remainder of this watcher run
	}
}

func (m *ehentai) removeCurrentIndex(index int) {
	m.multiProxy.mutex.Lock()
	defer m.multiProxy.mutex.Unlock()

	for i, v := range m.multiProxy.currentIndexes {
		if v == index {
			m.multiProxy.currentIndexes = append(m.multiProxy.currentIndexes[:i], m.multiProxy.currentIndexes[i+1:]...)
			break
		}
	}
}

func (m *ehentai) processDownloadQueueMultiProxy(downloadQueue []*imageGalleryItem, trackedItem *models.TrackedItem, notifications ...*models.Notification) error {
	slog.Info(fmt.Sprintf("found %d new items for uri: \"%s\"", len(downloadQueue), trackedItem.URI), "module", m.Key)

	for _, notification := range notifications {
		slog.Log(context.Background(),
			notification.Level, notification.Message, "module", m.Key)
	}

	for index, data := range downloadQueue {
		// wait until we have a free proxy, a proxy error, or every proxy exhausted its quota
		for !m.hasFreeProxy() && m.getProxyError() == nil && !m.allProxiesLimitReached() {
			time.Sleep(time.Millisecond * 100)
		}

		// if every proxy has hit its download limit, stop scheduling more work
		if m.allProxiesLimitReached() {
			slog.Info("download limit reached across all proxies, skipping galleries from now on", "module", m.Key)
			m.downloadLimitReached = true
			break
		}

		// handle if errors occurred in previous downloads
		if erroneousProxy := m.getProxyError(); erroneousProxy != nil {
			slog.Warn(fmt.Sprintf("error occurred during download for proxy: %s",
				erroneousProxy.proxy.Host), "module", m.Key)
			m.multiProxy.waitGroup.Wait()
			return erroneousProxy.occurredError
		}

		proxy := m.acquireFreeProxy()
		if proxy == nil {
			// another goroutine beat us to the last free proxy; retry this index in the next iteration
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

	// handle if errors occurred in previous downloads
	if erroneousProxy := m.getProxyError(); erroneousProxy != nil {
		slog.Warn(fmt.Sprintf("error occurred during download for proxy: %s",
			erroneousProxy.proxy.Host), "module", m.Key)
		return erroneousProxy.occurredError
	}

	if m.downloadLimitReached {
		return fmt.Errorf("download limit reached across all proxies")
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
	defer m.multiProxy.waitGroup.Done()

	currentSession := downloadSession
	defer func() {
		currentSession.inUse = false
	}()

	for {
		downloadQueueItem, err := m.getDownloadQueueItem(currentSession.session, trackedItem, data)
		if err != nil {
			currentSession.occurredError = err
			return
		}

		// an empty file URI typically means the proxy got rate-limited and the image page
		// no longer rendered the expected img tag - swap proxies rather than failing the item
		if downloadQueueItem.FileURI == "" {
			slog.Warn(fmt.Sprintf(
				"empty download URI for proxy: %s (likely rate-limited), excluding from rotation",
				currentSession.proxy.Host), "module", m.Key)
			currentSession.limitReached = true
			currentSession.inUse = false

			nextSession := m.waitForFreeProxy()
			if nextSession == nil {
				m.downloadLimitReached = true
				m.removeCurrentIndex(index)
				return
			}

			currentSession = nextSession
			continue
		}

		downloadErr := m.downloadImageSession(currentSession, trackedItem, downloadQueueItem, index)
		if downloadErr != nil && currentSession.limitReached {
			// the proxy exhausted its quota - release it and try the remaining proxies
			currentSession.inUse = false

			nextSession := m.waitForFreeProxy()
			if nextSession == nil {
				m.downloadLimitReached = true
				m.removeCurrentIndex(index)
				return
			}

			currentSession = nextSession
			continue
		}

		if downloadErr != nil {
			if downloadQueueItem.FallbackFileURI == "" {
				currentSession.occurredError = downloadErr
				return
			}

			// we have a fallback URI
			data.uri = downloadQueueItem.FallbackFileURI
			fallback, fallbackErr := m.getDownloadQueueItem(currentSession.session, trackedItem, data)
			if fallbackErr != nil {
				currentSession.occurredError = fallbackErr
				return
			}

			slog.Warn(fmt.Sprintf("received status code 404 on gallery url \"%s\", trying fallback url \"%s\"",
				data.uri,
				fallback.FileURI), "module", m.Key)

			downloadQueueItem.FileURI = fallback.FileURI
			downloadQueueItem.FallbackFileURI = ""

			// retry the fallback once and override the previous error with the new result
			currentSession.occurredError = m.downloadImageSession(currentSession, trackedItem, downloadQueueItem, index)
		}

		return
	}
}

func (m *ehentai) downloadImageSession(
	downloadSession *proxySession, trackedItem *models.TrackedItem, downloadQueueItem *models.DownloadQueueItem, index int,
) error {
	// check for per-proxy download limit
	if downloadQueueItem.FileURI == "https://exhentai.org/img/509.gif" ||
		downloadQueueItem.FileURI == "https://e-hentai.org/img/509.gif" {
		slog.Warn(fmt.Sprintf(
			"download limit reached for proxy: %s, excluding from rotation",
			downloadSession.proxy.Host), "module", m.Key)
		downloadSession.limitReached = true

		return fmt.Errorf("download limit reached for proxy %s", downloadSession.proxy.Host)
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
		if m.isLowestIndex(index) {
			// if we are the lowest index (to prevent skips on errors) update the downloaded item
			m.DbIO.UpdateTrackedItem(trackedItem, downloadQueueItem.ItemID)
		}

		m.removeCurrentIndex(index)
	}

	return downloadErr
}
