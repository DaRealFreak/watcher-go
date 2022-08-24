package ehentai

import (
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/DaRealFreak/watcher-go/internal/http"
	"github.com/DaRealFreak/watcher-go/internal/http/session"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/time/rate"
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
		singleSession.RateLimiter = rate.NewLimiter(rate.Every(2000*time.Millisecond), 1)
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

func (m *ehentai) processDownloadQueueMultiProxy(downloadQueue []*imageGalleryItem, trackedItem *models.TrackedItem) error {
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

	// if no error occurred update the tracked item to the last item ID
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
			fp.TruncateMaxLength(fp.SanitizePath(strings.TrimSpace(trackedItem.SubFolder), false)),
			fp.TruncateMaxLength(strings.TrimSpace(downloadQueueItem.DownloadTag)),
			fp.TruncateMaxLength(strings.TrimSpace(downloadQueueItem.FileName)),
		),
		downloadQueueItem.FileURI,
	)

	// if error == 404
	// img normally has onerror tag: onerror="this.onerror=null; nl('43323-460857')"
	// -> current url + "?nl=43323-460857

	if downloadSession.occurredError == nil {
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

	m.multiProxy.waitGroup.Done()
}
