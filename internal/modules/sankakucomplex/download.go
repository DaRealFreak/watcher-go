package sankakucomplex

import (
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/DaRealFreak/watcher-go/internal/http/session"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
	log "github.com/sirupsen/logrus"
)

type downloadQueue struct {
	items []*downloadGalleryItem
	books []*downloadBookItem
}

type downloadBookItem struct {
	bookId       string
	bookName     string
	bookLanguage []string
	items        []*downloadGalleryItem
}

type downloadGalleryItem struct {
	item *models.DownloadQueueItem
}

func (m *sankakuComplex) downloadDownloadQueueItem(trackedItem *models.TrackedItem, item *models.DownloadQueueItem) (bool, error) {
	u, err := url.Parse(item.FileURI)
	if err != nil {
		return false, err
	}

	parsedQuery, _ := url.ParseQuery(u.RawQuery)
	expiration := int64(0)
	if expires := parsedQuery.Get("e"); expires != "" {
		expiration, err = strconv.ParseInt(expires, 10, 64)
		if err != nil {
			return false, err
		}

		if time.Now().Unix() >= expiration {
			log.WithField("module", m.Key).Info(
				fmt.Sprintf(
					"links expired for uri, refreshing progress: \"%s\"",
					trackedItem.URI,
				),
			)

			return true, m.Parse(trackedItem)
		}
	}

	err = m.Session.DownloadFile(
		path.Join(m.GetDownloadDirectory(), m.Key, item.DownloadTag, item.FileName),
		item.FileURI,
	)

	if err != nil && item.FallbackFileURI != "" && item.FallbackFileURI != item.FileURI {
		// fallback to resized image
		log.WithField("module", m.Key).Warn(
			fmt.Sprintf("error occured: %s, using fallback URI", err.Error()),
		)

		item.FileURI = item.FallbackFileURI
		ext := fp.GetFileExtension(item.FileName)
		fileName := strings.TrimRight(item.FileName, ext)
		item.FileName = fmt.Sprintf("%s_fallback%s", fileName, ext)

		return m.downloadDownloadQueueItem(trackedItem, item)
	}

	if expiration > 0 && time.Now().Unix() >= expiration {
		log.WithField("module", m.Key).Info(
			fmt.Sprintf(
				"links expired for uri, refreshing progress: \"%s\"",
				trackedItem.URI,
			),
		)

		return true, m.Parse(trackedItem)
	}

	if e, ok := err.(session.StatusError); ok && e.StatusCode == 404 &&
		item.FallbackFileURI == item.FileURI &&
		regexp.MustCompile(`/books\?`).MatchString(trackedItem.URI) {
		log.WithField("module", m.Key).Warnf("skipping book item: %s, status code was 404", item.ItemID)
		return false, nil
	}

	return false, err
}

func (m *sankakuComplex) processDownloadQueue(downloadQueue *downloadQueue, trackedItem *models.TrackedItem) error {
	log.WithField("module", m.Key).Info(
		fmt.Sprintf("found %d new items for uri: \"%s\"", len(downloadQueue.items)+len(downloadQueue.books), trackedItem.URI),
	)

	for index, data := range downloadQueue.items {
		log.WithField("module", m.Key).Info(
			fmt.Sprintf(
				"downloading updates for uri: \"%s\" (%0.2f%%)",
				trackedItem.URI,
				float64(index+1)/float64(len(downloadQueue.items))*100,
			),
		)

		if expired, err := m.downloadDownloadQueueItem(trackedItem, data.item); expired || err != nil {
			if err != nil {
				if e, ok := err.(session.StatusError); ok && e.StatusCode == 404 {
					// continue on 404 errors, since they most likely won't get fixed
					log.WithField("module", m.Key).Warnf("skipping item: %s, status code was 404", data.item.ItemID)
				} else if m.settings.Download.SkipBrokenStreams && strings.HasPrefix(err.Error(), "stream error: stream ID") {
					// skip broken streams if configured to do so
					log.WithField("module", m.Key).Warn(
						fmt.Sprintf("skipping item: %s, download stream is broken", data.item.ItemID),
					)
				} else {
					return err
				}
			}
			// on no error we still break the download queue after we ran into expired links
			if expired {
				break
			}
		}

		m.DbIO.UpdateTrackedItem(trackedItem, data.item.ItemID)
	}

	for index, data := range downloadQueue.books {
		tagName, err := m.extractItemTag(trackedItem)
		if err != nil {
			return err
		}

		log.WithField("module", m.Key).Info(
			fmt.Sprintf(
				"downloading updates for uri: \"%s\" (%0.2f%%)",
				trackedItem.URI,
				float64(index+1)/float64(len(downloadQueue.books))*100,
			),
		)

		bookLanguage := ""
		if len(data.bookLanguage) > 0 {
			bookLanguage = fmt.Sprintf(" [%s]", strings.Join(data.bookLanguage, ", "))
		}

		for i, singleItem := range data.items {
			singleItem.item.FileName = fmt.Sprintf("%d_%s", i+1, singleItem.item.FileName)
			singleItem.item.DownloadTag = fmt.Sprintf("%s/%s/%s",
				fp.SanitizePath(tagName, false),
				"books",
				fmt.Sprintf("%s%s (%s)", fp.SanitizePath(data.bookName, false), bookLanguage, data.bookId),
			)

			if expired, downloadErr := m.downloadDownloadQueueItem(trackedItem, singleItem.item); expired || downloadErr != nil {
				if downloadErr != nil {
					return downloadErr
				}
				// on no error we still break the download queue after we ran into expired links
				if expired {
					return m.Parse(trackedItem)
				}
			}
		}

		m.DbIO.UpdateTrackedItem(trackedItem, data.bookId)
	}

	return nil
}
