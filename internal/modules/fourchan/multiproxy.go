package fourchan

import (
	"fmt"
	"github.com/DaRealFreak/watcher-go/internal/http"
	"github.com/DaRealFreak/watcher-go/internal/http/session"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
	log "github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
	"net/url"
	"os"
	"path"
	"strings"
	"time"
)

type proxySession struct {
	inUse         bool
	proxy         http.ProxySettings
	session       *session.DefaultSession
	occurredError error
}

func (m *fourChan) initializeProxySessions() {
	// copy the cookies for 4chan to desuarchive
	fourChanUrl, _ := url.Parse("https://www.4chan.org/")
	archiveUrl, _ := url.Parse("https://desuarchive.org/")

	for _, proxy := range m.settings.LoopProxies {
		if proxy.Enable {
			singleSession := session.NewSession(m.Key)
			singleSession.RateLimiter = rate.NewLimiter(rate.Every(time.Duration(m.rateLimit)*time.Millisecond), 1)
			// copy login cookies for session
			singleSession.Client.SetCookies(fourChanUrl, m.Session.GetClient().GetCookies(fourChanUrl))
			singleSession.Client.SetCookies(archiveUrl, m.Session.GetClient().GetCookies(fourChanUrl))
			raven.CheckError(singleSession.SetProxy(&proxy))
			m.proxies = append(m.proxies, &proxySession{
				inUse:         false,
				proxy:         proxy,
				session:       singleSession,
				occurredError: nil,
			})
		}
	}
}

func (m *fourChan) isLowestIndex(index int) bool {
	lowestIndex := 999
	for _, v := range m.multiProxy.currentIndexes {
		if v < lowestIndex {
			lowestIndex = v
		}
	}

	return lowestIndex == index
}

func (m *fourChan) hasFreeProxy() bool {
	for _, proxy := range m.proxies {
		if !proxy.inUse {
			return true
		}
	}

	return false
}

func (m *fourChan) getFreeProxy() *proxySession {
	for _, proxy := range m.proxies {
		if !proxy.inUse {
			return proxy
		}
	}

	return nil
}

func (m *fourChan) getProxyError() *proxySession {
	for _, proxy := range m.proxies {
		if proxy.occurredError != nil {
			return proxy
		}
	}

	return nil
}

func (m *fourChan) resetProxies() {
	for _, proxy := range m.proxies {
		proxy.inUse = false
		proxy.occurredError = nil
	}
}

func (m *fourChan) processDownloadQueueMultiProxy(downloadQueue []models.DownloadQueueItem, trackedItem *models.TrackedItem) error {
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

	// if no error occurred update the tracked item to the last item ID
	if len(downloadQueue) > 0 {
		m.DbIO.UpdateTrackedItem(trackedItem, downloadQueue[len(downloadQueue)-1].ItemID)
	}

	return nil
}

func (m *fourChan) downloadItemSession(
	downloadSession *proxySession, trackedItem *models.TrackedItem, downloadQueueItem models.DownloadQueueItem, index int,
) {
	if err := m.downloadImageSession(downloadSession, trackedItem, downloadQueueItem, index); err != nil {
		if downloadQueueItem.FallbackFileURI == "" {
			downloadSession.occurredError = err
			downloadSession.inUse = false
			m.multiProxy.waitGroup.Done()
			return
		}

		log.WithField("module", m.Key).Warnf(
			"received status code 404 on gallery url \"%s\"",
			downloadQueueItem.FileURI,
		)
	}

	downloadSession.inUse = false
	m.multiProxy.waitGroup.Done()
}

func (m *fourChan) downloadImageSession(
	downloadSession *proxySession, trackedItem *models.TrackedItem, downloadQueueItem models.DownloadQueueItem, index int,
) error {
	startTime := time.Now()

	// apply rate limit for the current session since we don't use the wrapper function
	downloadSession.session.ApplyRateLimit()

	// directly request the file URI
	res, err := downloadSession.session.GetClient().Get(downloadQueueItem.FileURI)
	if err != nil {
		return err
	}

	if res.StatusCode == 429 {
		log.WithField("module", m.Key).Warnf(
			"received status code 429 on gallery url \"%s\"",
			downloadQueueItem.FileURI,
		)
		time.Sleep(time.Second * 5)

		return m.downloadImageSession(downloadSession, trackedItem, downloadQueueItem, index)
	}

	if res.StatusCode != 404 {
		dst := path.Join(
			m.GetDownloadDirectory(),
			m.Key,
			fp.TruncateMaxLength(fp.SanitizePath(strings.TrimSpace(trackedItem.SubFolder), false)),
			fp.TruncateMaxLength(strings.TrimSpace(downloadQueueItem.DownloadTag)),
			fp.TruncateMaxLength(strings.TrimSpace(downloadQueueItem.FileName)),
		)
		downloadErr := downloadSession.session.DownloadFileFromResponse(res, dst)
		if downloadErr == nil {
			// bump the file’s mtime so that ordering by time == ordering by index
			if info, statErr := os.Stat(dst); statErr == nil && !info.IsDir() {
				if chtErr := os.Chtimes(dst, startTime, startTime); chtErr != nil {
					log.WithField("module", m.Key).Warnf(
						"failed to reset timestamp for %s: %v",
						dst, chtErr,
					)
				}
			}

			// back to original logic
			downloadSession.inUse = false

			if m.isLowestIndex(index) {
				// if we are the lowest index (to prevent skips on errors), update the downloaded item
				m.DbIO.UpdateTrackedItem(trackedItem, downloadQueueItem.ItemID)
			}

			// remove the current index from the current list since we finished
			for i, v := range m.multiProxy.currentIndexes {
				if v == index {
					m.multiProxy.currentIndexes = append(m.multiProxy.currentIndexes[:i], m.multiProxy.currentIndexes[i+1:]...)
					break
				}
			}
		}

		return downloadErr
	} else {
		log.WithField("module", m.Key).Warnf(
			"received status code 404 on gallery url \"%s\"",
			downloadQueueItem.FileURI,
		)

		// it's completely normal for 404 errors to occur on that website. the image just doesn't exist anymore,
		// so log a warning and return nil
		return nil
	}
}
