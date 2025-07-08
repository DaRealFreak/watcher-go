package deviantart

import (
	"errors"
	"fmt"
	"github.com/DaRealFreak/watcher-go/internal/http"
	"github.com/DaRealFreak/watcher-go/internal/http/tls_session"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules/deviantart/napi"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	log "github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
	"io"
	"net/url"
	"time"
)

type proxySession struct {
	inUse         bool
	proxy         http.ProxySettings
	session       *tls_session.TlsClientSession
	occurredError error
}

func (m *deviantArt) initializeProxySessions() {
	// copy the cookies from the logged-in session
	daURL, _ := url.Parse("https://deviantart.com")
	daWwwURL, _ := url.Parse("https://www.deviantart.com")

	for _, proxy := range m.settings.LoopProxies {
		singleSession := tls_session.NewTlsClientSession(m.Key, napi.DeviantArtErrorHandler{ModuleKey: m.ModuleKey()})
		singleSession.RateLimiter = rate.NewLimiter(rate.Every(time.Duration(m.rateLimit)*time.Millisecond), 1)
		// copy login cookies for the session
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

func (m *deviantArt) isLowestIndex(index int) bool {
	lowestIndex := 999
	for _, v := range m.multiProxy.currentIndexes {
		if v < lowestIndex {
			lowestIndex = v
		}
	}

	return lowestIndex == index
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

			go m.downloadItemSessionNapi(proxy, trackedItem, data, index)

			// sleep 100 milliseconds to queue the next download after the current one
			time.Sleep(time.Millisecond * 100)
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
		m.DbIO.UpdateTrackedItem(trackedItem, downloadQueue[len(downloadQueue)-1].itemID)
	}

	return nil
}

func (m *deviantArt) downloadItemSessionNapi(
	downloadSession *proxySession, trackedItem *models.TrackedItem, deviationItem downloadQueueItemNAPI, index int,
) {
	downloadSession.occurredError = m.downloadDeviationNapi(trackedItem, deviationItem, downloadSession.session, false)

	if downloadSession.occurredError == nil {
		if m.isLowestIndex(index) {
			// if we are the lowest index (to prevent skips on errors), update the downloaded item
			m.DbIO.UpdateTrackedItem(trackedItem, deviationItem.itemID)
		}

		// remove the current index from the current list since we finished
		for i, v := range m.multiProxy.currentIndexes {
			if v == index {
				m.multiProxy.currentIndexes = append(m.multiProxy.currentIndexes[:i], m.multiProxy.currentIndexes[i+1:]...)
				break
			}
		}
	} else {
		log.WithField("module", m.Key).Errorf(
			"error occurred downloading item %s (%s) with proxy %s: %s",
			trackedItem.URI, deviationItem.itemID, downloadSession.proxy.Host, downloadSession.occurredError.Error(),
		)

		var scErr tls_session.StatusError
		if errors.As(downloadSession.occurredError, &scErr) {
			// 404 and 400 errors are mostly caused by expired CSRF tokens, not exactly sure where to refresh it
			// reading the home page returns 200 and a CSRF token, but it's invalid for the existing queue
			if scErr.StatusCode == 404 || scErr.StatusCode == 400 {
				res, err2 := m.nAPI.UserSession.Get(deviationItem.deviation.URL)
				if err2 != nil {
					return
				}

				html, _ := io.ReadAll(res.Body)
				_ = html
			}
		}
	}

	downloadSession.inUse = false
	m.multiProxy.waitGroup.Done()
}
