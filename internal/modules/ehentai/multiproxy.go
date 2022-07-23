package ehentai

import (
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"

	"golang.org/x/time/rate"

	"github.com/DaRealFreak/watcher-go/internal/models"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/DaRealFreak/watcher-go/internal/http"
	"github.com/DaRealFreak/watcher-go/internal/http/session"
	"github.com/DaRealFreak/watcher-go/internal/raven"
)

type proxySession struct {
	inUse         bool
	proxy         http.ProxySettings
	session       *session.DefaultSession
	occurredError error
}

func (m *ehentai) initializeProxySessions() {
	// copy the cookies for e-hentai to exhentai
	ehURL, _ := url.Parse("https://e-hentai.org")
	exURL, _ := url.Parse("https://exhentai.org")

	for _, proxy := range m.settings.LoopProxies {
		singleSession := session.NewSession(m.Key)
		singleSession.RateLimiter = rate.NewLimiter(rate.Every(1500*time.Millisecond), 1)
		// copy login cookies for session
		singleSession.Client.Jar.SetCookies(ehURL, m.Session.GetClient().Jar.Cookies(ehURL))
		singleSession.Client.Jar.SetCookies(exURL, m.Session.GetClient().Jar.Cookies(ehURL))
		raven.CheckError(singleSession.SetProxy(&proxy))
		m.proxies = append(m.proxies, &proxySession{
			inUse:         false,
			proxy:         proxy,
			session:       singleSession,
			occurredError: nil,
		})
	}
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

func (m *ehentai) getProxyError() error {
	for _, proxy := range m.proxies {
		if proxy.occurredError != nil {
			return proxy.occurredError
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

func (m *ehentai) processDownloadQueueMultiProxy(downloadQueue []imageGalleryItem, trackedItem *models.TrackedItem) error {
	log.WithField("module", m.Key).Info(
		fmt.Sprintf("found %d new items for uri: \"%s\"", len(downloadQueue), trackedItem.URI),
	)

	for index, data := range downloadQueue {
		// sleep until we have a free proxy again
		for !m.hasFreeProxy() && m.getProxyError() == nil {
			time.Sleep(time.Millisecond * 100)
			continue
		}

		// handle if errors occurred in previous downloads
		if err := m.getProxyError(); err != nil {
			return m.getProxyError()
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

			go m.downloadItemSession(proxy, trackedItem, data)
		}

	}

	m.multiProxy.waitGroup.Wait()

	// handle if errors occurred in previous downloads
	if err := m.getProxyError(); err != nil {
		return m.getProxyError()
	}

	return nil
}

func (m *ehentai) downloadItemSession(
	downloadSession *proxySession, trackedItem *models.TrackedItem, data imageGalleryItem,
) {
	downloadQueueItem, err := m.getDownloadQueueItem(downloadSession.session, trackedItem, data)
	if err != nil {
		downloadSession.occurredError = err
		downloadSession.inUse = false
		m.multiProxy.waitGroup.Done()
		return
	}

	// check for limit
	if downloadQueueItem.FileURI == "https://exhentai.org/img/509.gif" ||
		downloadQueueItem.FileURI == "https://e-hentai.org/img/509.gif" {
		log.WithField("module", m.Key).Info("download limit reached, skipping galleries from now on")
		m.downloadLimitReached = true

		downloadSession.occurredError = fmt.Errorf("download limit reached")
		m.multiProxy.waitGroup.Done()
		return
	}

	downloadSession.occurredError = downloadSession.session.DownloadFile(
		path.Join(
			viper.GetString("download.directory"),
			m.Key,
			m.TruncateMaxLength(strings.TrimSpace(downloadQueueItem.DownloadTag)),
			m.TruncateMaxLength(strings.TrimSpace(downloadQueueItem.FileName)),
		),
		downloadQueueItem.FileURI,
	)

	if downloadSession.occurredError == nil {
		downloadSession.inUse = false
	}

	m.multiProxy.waitGroup.Done()
}
