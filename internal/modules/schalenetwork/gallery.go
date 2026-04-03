package schalenetwork

import (
	"fmt"
	"log/slog"
	"path"
	"strconv"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
)

// parseGallery parses the tracked item if we detected a tracked gallery
func (m *schaleNetwork) parseGallery(item *models.TrackedItem) error {
	matches := m.galleryPattern.FindStringSubmatch(item.URI)
	if len(matches) < 3 {
		return fmt.Errorf("could not extract gallery id and key from URI: %s", item.URI)
	}

	id := matches[1]
	key := matches[2]

	detail, err := m.getBookDetail(id, key)
	if err != nil {
		return err
	}

	return m.parseGalleryFromDetail(item, detail, id, key)
}

// parseGalleryFromDetail parses a gallery using its detail response
func (m *schaleNetwork) parseGalleryFromDetail(
	item *models.TrackedItem,
	detail *bookDetailResponse,
	id, key string,
) error {
	bookData, err := m.getBookData(id, key)
	if err != nil {
		return err
	}

	format, fmtW, err := m.selectFormat(bookData)
	if err != nil {
		return fmt.Errorf("gallery %s/%s: %w", id, key, err)
	}

	imageList, err := m.getBookImages(id, key, format.ID.String(), format.Key, fmtW)
	if err != nil {
		return err
	}

	galleryItems := m.getGalleryImageURLs(detail, imageList)

	// iterate backwards over the items to be able to break on the first match
	var downloadQueue []models.DownloadQueueItem
	for i := len(galleryItems) - 1; i >= 0; i-- {
		galleryItem := galleryItems[i]

		if item.CurrentItem == galleryItem.ItemID {
			break
		}

		downloadQueue = append(downloadQueue, galleryItem)
	}

	// reverse download queue to download oldest items first
	for i, j := 0, len(downloadQueue)-1; i < j; i, j = i+1, j-1 {
		downloadQueue[i], downloadQueue[j] = downloadQueue[j], downloadQueue[i]
	}

	if m.settings.MultiProxy && len(m.proxies) > 0 {
		m.resetProxies()
		if err = m.processDownloadQueueMultiProxy(downloadQueue, item); err != nil {
			return err
		}
	} else {
		if err = m.processDownloadQueue(downloadQueue, item); err != nil {
			return err
		}
	}

	m.DbIO.ChangeTrackedItemCompleteStatus(item, true)

	return nil
}

// processDownloadQueue downloads files with the required headers for the CDN
func (m *schaleNetwork) processDownloadQueue(downloadQueue []models.DownloadQueueItem, trackedItem *models.TrackedItem) error {
	slog.Info(
		fmt.Sprintf("found %d new items for uri: \"%s\"", len(downloadQueue), trackedItem.URI),
		"module", m.Key,
	)

	for index, data := range downloadQueue {
		slog.Info(
			fmt.Sprintf(
				"downloading updates for uri: \"%s\" (%0.2f%%)",
				trackedItem.URI,
				float64(index+1)/float64(len(downloadQueue))*100,
			),
			"module", m.Key,
		)

		filePath := path.Join(
			m.GetDownloadDirectory(),
			m.Key,
			fp.TruncateMaxLength(fp.SanitizePath(trackedItem.SubFolder, false)),
			fp.TruncateMaxLength(fp.SanitizePath(data.DownloadTag, false)),
			fp.TruncateMaxLength(fp.SanitizePath(data.FileName, false)),
		)

		resp, err := m.get(data.FileURI)
		if err != nil {
			return err
		}

		if err = m.Session.DownloadFileFromResponse(resp, filePath); err != nil {
			return err
		}

		m.DbIO.UpdateTrackedItem(trackedItem, data.ItemID)
	}

	return nil
}

// selectFormat selects the best available format from the book data
func (m *schaleNetwork) selectFormat(bookData *bookDataResponse) (bookFormat, string, error) {
	formatPriority := []string{"0", "1600", "1280", "980", "780"}

	for _, fmtW := range formatPriority {
		if f, ok := bookData.Data[fmtW]; ok && f.ID.String() != "" {
			slog.Debug(fmt.Sprintf("selected format %s", fmtW), "module", m.Key)
			return f, fmtW, nil
		}
	}

	return bookFormat{}, "", fmt.Errorf("no available format found")
}

// getGalleryImageURLs constructs download queue items from the book detail and image list
func (m *schaleNetwork) getGalleryImageURLs(
	detail *bookDetailResponse,
	imageList *bookImageListResponse,
) []models.DownloadQueueItem {
	language := detail.GetLanguage()

	var downloadTag string
	if language != "" {
		downloadTag = fmt.Sprintf("%s [%s] (%d)",
			fp.SanitizePath(detail.GetTitle(), false),
			language,
			detail.ID,
		)
	} else {
		downloadTag = fmt.Sprintf("%s (%d)",
			fp.SanitizePath(detail.GetTitle(), false),
			detail.ID,
		)
	}

	items := make([]models.DownloadQueueItem, 0, len(imageList.Entries))
	for i, entry := range imageList.Entries {
		fileURI := imageList.Base + entry.Path

		items = append(items, models.DownloadQueueItem{
			ItemID:      strconv.Itoa(i + 1),
			DownloadTag: downloadTag,
			FileName:    fmt.Sprintf("%03d_%s", i+1, fp.GetFileName(fileURI)),
			FileURI:     fileURI,
		})
	}

	return items
}
