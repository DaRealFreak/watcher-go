package sankakucomplex

import (
	"fmt"
	"github.com/DaRealFreak/watcher-go/internal/modules/sankakucomplex/api"
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
	bookApiItem  *api.BookApiItem
}

type downloadGalleryItem struct {
	item               *models.DownloadQueueItem
	apiData            *api.ApiItem
	singleBookDownload bool
}

func (m *sankakuComplex) downloadDownloadQueueItem(trackedItem *models.TrackedItem, galleryItem *downloadGalleryItem) (bool, error) {
	u, err := url.Parse(galleryItem.item.FileURI)
	if err != nil {
		return false, err
	}

	bookResponse, bookErr := m.api.GetPoolResponse(galleryItem.apiData.ID)
	if bookErr != nil {
		return false, bookErr
	}

	if bookResponse != nil && bookResponse.Name != "" {
		if !strings.Contains(galleryItem.item.DownloadTag, "/books/") && !galleryItem.singleBookDownload {
			bookData, bookDataErr := m.api.GetBookResponse(bookResponse.ID)
			if bookDataErr != nil {
				return false, bookDataErr
			}

			postBookIndex := bookData.GetPostIndex(galleryItem.apiData.ID)

			idString := fmt.Sprintf(" (%v)", bookResponse.ID)
			galleryItem.item.FileName = fmt.Sprintf(
				"%d_%s",
				postBookIndex+1,
				galleryItem.item.FileName,
			)
			galleryItem.item.DownloadTag = path.Join(
				galleryItem.item.DownloadTag,
				"books",
				fmt.Sprintf("%s%s",
					fp.TruncateMaxLength(
						fp.SanitizePath(bookResponse.Name, false),
						len(idString),
					),
					idString,
				),
			)
		}
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
		path.Join(m.GetDownloadDirectory(), m.Key, galleryItem.item.DownloadTag, galleryItem.item.FileName),
		galleryItem.item.FileURI,
	)

	if err != nil && galleryItem.item.FallbackFileURI != "" && galleryItem.item.FallbackFileURI != galleryItem.item.FileURI {
		// fallback to resized image
		log.WithField("module", m.Key).Warn(
			fmt.Sprintf("error occurred: %s, using fallback URI", err.Error()),
		)

		galleryItem.item.FileURI = galleryItem.item.FallbackFileURI
		ext := fp.GetFileExtension(galleryItem.item.FileName)
		fileName := strings.TrimRight(galleryItem.item.FileName, ext)
		galleryItem.item.FileName = fmt.Sprintf("%s_fallback%s", fileName, ext)

		return m.downloadDownloadQueueItem(trackedItem, galleryItem)
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
		galleryItem.item.FallbackFileURI == galleryItem.item.FileURI &&
		regexp.MustCompile(`/books\?`).MatchString(trackedItem.URI) {
		log.WithField("module", m.Key).Warnf("skipping book galleryItem: %s, status code was 404", galleryItem.item.ItemID)
		return false, nil
	}

	return false, err
}

func (m *sankakuComplex) processDownloadQueue(downloadQueue *downloadQueue, trackedItem *models.TrackedItem, notifications ...*models.Notification) error {
	log.WithField("module", m.Key).Info(
		fmt.Sprintf("found %d new items for uri: \"%s\"", len(downloadQueue.items)+len(downloadQueue.books), trackedItem.URI),
	)

	for _, notification := range notifications {
		log.WithField("module", m.Key).Log(
			notification.Level,
			notification.Message,
		)
	}

	for index, data := range downloadQueue.items {
		log.WithField("module", m.Key).Info(
			fmt.Sprintf(
				"downloading updates for uri: \"%s\" (%0.2f%%)",
				trackedItem.URI,
				float64(index+1)/float64(len(downloadQueue.items))*100,
			),
		)

		if expired, err := m.downloadDownloadQueueItem(trackedItem, data); expired || err != nil {
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
			// remove book languages already in the book name
			var cleanedLanguages []string
			for _, lang := range data.bookLanguage {
				if !strings.Contains(strings.ToLower(data.bookName), strings.ToLower(lang)) {
					cleanedLanguages = append(cleanedLanguages, lang)
				}
			}
			// only add the book language if we have any left
			if len(cleanedLanguages) > 0 {
				bookLanguage = fmt.Sprintf(" [%s]", strings.Join(cleanedLanguages, ", "))
			}
		}

		tmpDownloadQueue, err := m.extractBookItems(data.bookApiItem)
		if err != nil {
			return err
		}

		for i, singleItem := range tmpDownloadQueue {
			idString := fmt.Sprintf("%s (%s)", bookLanguage, data.bookApiItem.ID)
			bookFolder := fmt.Sprintf("%s%s",
				fp.TruncateMaxLength(
					fp.SanitizePath(data.bookName, false),
					len(idString),
				),
				idString,
			)

			singleItem.item.FileName = fmt.Sprintf("%d_%s", i+1, singleItem.item.FileName)
			singleItem.item.DownloadTag = fmt.Sprintf("%s/%s/%s",
				fp.SanitizePath(tagName, false),
				"books",
				bookFolder,
			)

			if expired, downloadErr := m.downloadDownloadQueueItem(trackedItem, singleItem); expired || downloadErr != nil {
				if downloadErr != nil {
					return downloadErr
				}
				// on no error we still break the download queue after we ran into expired links
				return m.Parse(trackedItem)
			}
		}

		m.DbIO.UpdateTrackedItem(trackedItem, data.bookId)
	}

	return nil
}
