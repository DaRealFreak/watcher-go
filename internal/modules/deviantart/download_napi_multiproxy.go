package deviantart

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"context"
	"log/slog"

	"github.com/DaRealFreak/watcher-go/internal/http"
	"github.com/DaRealFreak/watcher-go/internal/http/tls_session"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules/deviantart/napi"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"golang.org/x/time/rate"
)

type proxySession struct {
	inUse         bool
	proxy         http.ProxySettings
	session       *tls_session.TlsClientSession
	occurredError error
}

func (m *deviantArt) initializeProxySessions() {
	// get the shared cookie jar from the main session so all proxy sessions
	// share the same cookies and stay in sync with Set-Cookie updates
	mainSession, ok := m.nAPI.UserSession.(*tls_session.TlsClientSession)
	if !ok {
		slog.Error("cannot share cookie jar: main session is not a TlsClientSession, falling back to cookie copy")
		m.initializeProxySessionsLegacy()
		return
	}

	sharedJar := mainSession.Jar

	for _, proxy := range m.settings.LoopProxies {
		if !proxy.Enable {
			continue
		}

		singleSession := tls_session.NewTlsClientSessionWithJar(m.Key, sharedJar, napi.DeviantArtErrorHandler{ModuleKey: m.ModuleKey()})
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

// initializeProxySessionsLegacy is the fallback that copies cookies once (old behavior)
func (m *deviantArt) initializeProxySessionsLegacy() {
	daURL, _ := url.Parse("https://deviantart.com")
	daWwwURL, _ := url.Parse("https://www.deviantart.com")

	for _, proxy := range m.settings.LoopProxies {
		if !proxy.Enable {
			continue
		}

		singleSession := tls_session.NewTlsClientSession(m.Key, napi.DeviantArtErrorHandler{ModuleKey: m.ModuleKey()})
		singleSession.RateLimiter = rate.NewLimiter(rate.Every(time.Duration(m.rateLimit)*time.Millisecond), 1)
		singleSession.Client.SetCookies(daWwwURL, m.nAPI.UserSession.GetClient().GetCookies(daWwwURL))
		singleSession.Client.SetCookies(daURL, m.nAPI.UserSession.GetClient().GetCookies(daURL))
		raven.CheckError(singleSession.SetProxy(&proxy))
		m.proxies = append(m.proxies, &proxySession{
			inUse:         false,
			proxy:         proxy,
			session:       singleSession,
			occurredError: nil,
		})
	}
}

func (m *deviantArt) hasFreeProxy() bool {
	for _, proxy := range m.proxies {
		if !proxy.inUse {
			return true
		}
	}

	return false
}

func (m *deviantArt) getFreeProxy() *proxySession {
	for _, proxy := range m.proxies {
		if !proxy.inUse {
			return proxy
		}
	}

	return nil
}

func (m *deviantArt) getProxyError() *proxySession {
	for _, proxy := range m.proxies {
		if proxy.occurredError != nil {
			return proxy
		}
	}

	return nil
}

func (m *deviantArt) resetProxies() {
	for _, proxy := range m.proxies {
		proxy.inUse = false
		proxy.occurredError = nil
	}
}

func (m *deviantArt) processDownloadQueueMultiProxy(downloadQueue []downloadQueueItemNAPI, trackedItem *models.TrackedItem, notifications ...*models.Notification) error {
	slog.Info(fmt.Sprintf("found %d new items for uri: \"%s\"", len(downloadQueue), trackedItem.URI), "module", m.Key)

	for _, notification := range notifications {
		slog.Log(context.Background(),
			notification.Level, notification.Message, "module", m.Key)
	}

	// track which items completed successfully for progress saving on error
	completedItems := make([]bool, len(downloadQueue))

	for index, data := range downloadQueue {
		// sleep until we have a free proxy again
		for !m.hasFreeProxy() && m.getProxyError() == nil {
			time.Sleep(time.Millisecond * 100)
			continue
		}

		// handle if errors occurred in previous downloads
		if erroneousProxy := m.getProxyError(); erroneousProxy != nil {
			slog.Warn(fmt.Sprintf("error occurred during download for proxy: %s",
				erroneousProxy.proxy.Host), "module", m.Key)
			break
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

			go m.downloadItemSessionNapi(proxy, trackedItem, downloadQueue, data, index, completedItems)

			// sleep 100 milliseconds to queue the next download after the current one
			time.Sleep(time.Millisecond * 100)
		}

	}

	// always wait for in-flight goroutines before returning
	m.multiProxy.waitGroup.Wait()

	// final safety-net save (progress is also persisted incrementally as each item
	// completes in downloadItemSessionNapi). all goroutines have finished here, so no
	// locking is needed for the completedItems read.
	if lastCompleted := lastContiguousCompleted(completedItems); lastCompleted >= 0 {
		m.DbIO.UpdateTrackedItem(trackedItem, downloadQueue[lastCompleted].itemID)
	}

	// handle if errors occurred in previous downloads
	if erroneousProxy := m.getProxyError(); erroneousProxy != nil {
		slog.Warn(fmt.Sprintf("error occurred during download for proxy: %s",
			erroneousProxy.proxy.Host), "module", m.Key)
		return m.getProxyError().occurredError
	}

	return nil
}

func (m *deviantArt) downloadItemSessionNapi(
	downloadSession *proxySession, trackedItem *models.TrackedItem, downloadQueue []downloadQueueItemNAPI,
	deviationItem downloadQueueItemNAPI, index int, completedItems []bool,
) {
	downloadSession.occurredError = m.downloadDeviationNapi(trackedItem, deviationItem, downloadSession.session, false)

	if downloadSession.occurredError == nil {
		m.markCompletedAndPersist(trackedItem, downloadQueue, completedItems, index)

		// remove the current index from the current list since we finished
		for i, v := range m.multiProxy.currentIndexes {
			if v == index {
				m.multiProxy.currentIndexes = append(m.multiProxy.currentIndexes[:i], m.multiProxy.currentIndexes[i+1:]...)
				break
			}
		}
	} else {
		slog.Error(fmt.Sprintf("error occurred downloading item %s (%s) with proxy %s: %s (CSRF: %s)",
			trackedItem.URI, deviationItem.itemID, downloadSession.proxy.Host, downloadSession.occurredError.Error(), m.nAPI.CSRFToken), "module", m.Key)

		var scErr tls_session.StatusError
		if errors.As(downloadSession.occurredError, &scErr) {
			if scErr.StatusCode == 400 && strings.Contains(scErr.Body, "image is invalid") {
				// broken/corrupt image on DA's side, skip and continue
				slog.Warn(fmt.Sprintf("skipping invalid image for deviation %s (%s): %s",
					deviationItem.deviation.URL, deviationItem.itemID, scErr.Body), "module", m.Key)
				downloadSession.occurredError = nil
				m.markCompletedAndPersist(trackedItem, downloadQueue, completedItems, index)

				for i, v := range m.multiProxy.currentIndexes {
					if v == index {
						m.multiProxy.currentIndexes = append(m.multiProxy.currentIndexes[:i], m.multiProxy.currentIndexes[i+1:]...)
						break
					}
				}
			}
			// other 400s (invalid crop, expired token, etc.) and 404s bubble up
			// to processDownloadQueueNapi which triggers a re-login
		}
	}

	downloadSession.inUse = false
	m.multiProxy.waitGroup.Done()
}

// markCompletedAndPersist marks the item at index as completed and persists the tracked
// item's current_item up to the last contiguously completed item. Items complete out of
// order across proxies, so we only advance the watermark across a leading run of
// completed items - never skipping one that is still in flight. Safe to call
// concurrently from the proxy download goroutines.
func (m *deviantArt) markCompletedAndPersist(
	trackedItem *models.TrackedItem, downloadQueue []downloadQueueItemNAPI, completedItems []bool, index int,
) {
	m.multiProxy.progressMutex.Lock()
	defer m.multiProxy.progressMutex.Unlock()

	completedItems[index] = true

	if lastCompleted := lastContiguousCompleted(completedItems); lastCompleted >= 0 {
		m.DbIO.UpdateTrackedItem(trackedItem, downloadQueue[lastCompleted].itemID)
	}
}

// lastContiguousCompleted returns the index of the last item in the leading run of
// completed items (starting at 0), or -1 if the very first item is not yet complete.
// The download queue is ordered oldest-to-newest, so this is the furthest point up to
// which progress can be saved without skipping an unfinished item.
func lastContiguousCompleted(completedItems []bool) int {
	lastCompleted := -1
	for _, done := range completedItems {
		if !done {
			break
		}
		lastCompleted++
	}

	return lastCompleted
}
