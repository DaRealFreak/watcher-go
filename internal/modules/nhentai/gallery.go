package nhentai

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
	log "github.com/sirupsen/logrus"
)

func (m *nhentai) parseGalleryFromApiResponse(
	item *models.TrackedItem,
	apiResponse *galleryResponse,
) error {
	galleryTitle := fmt.Sprintf("%s [%s]", apiResponse.GetTitle(), apiResponse.GetLanguage())
	for _, blacklistedTag := range m.settings.Search.BlacklistedTags {
		if strings.Contains(strings.ToLower(galleryTitle), strings.ToLower(blacklistedTag)) {
			log.WithField("module", m.Key).Warnf(
				"gallery title \"%s\" contains blacklisted tag \"%s\", setting item to complete",
				galleryTitle,
				blacklistedTag,
			)
			m.DbIO.ChangeTrackedItemCompleteStatus(item, true)
			return nil
		}
	}

	galleryItems := m.getGalleryImageUrls(apiResponse)

	// iterate backwards over the items to be able to break on the first match
	var downloadQueue []models.DownloadQueueItem
	for i := len(galleryItems) - 1; i >= 0; i-- {
		galleryItem := galleryItems[i]

		// found current item, breaking
		if item.CurrentItem == galleryItem.ItemID {
			break
		}

		downloadQueue = append(downloadQueue, galleryItem)
	}

	// reverse download queue to download new items first
	for i, j := 0, len(downloadQueue)-1; i < j; i, j = i+1, j-1 {
		downloadQueue[i], downloadQueue[j] = downloadQueue[j], downloadQueue[i]
	}

	if err := m.ProcessDownloadQueue(downloadQueue, item); err != nil {
		return err
	}

	m.DbIO.ChangeTrackedItemCompleteStatus(item, true)

	return nil
}

// parseGallery parses the tracked item if we detected a tracked gallery
func (m *nhentai) parseGallery(item *models.TrackedItem) error {
	galleryId := m.searchGalleryIDPattern.FindStringSubmatch(item.URI)[1]
	apiResponse, err := m.getGallery(galleryId)
	if err != nil {
		return err
	}

	return m.parseGalleryFromApiResponse(item, apiResponse)
}

func (m *nhentai) getGalleryImageUrls(galleryResponse *galleryResponse) (galleryItems []models.DownloadQueueItem) {
	galleryLanguage := galleryResponse.GetLanguage()
	for i, imageURL := range galleryResponse.GetImages() {
		galleryItems = append(galleryItems, models.DownloadQueueItem{
			ItemID: strconv.Itoa(i + 1),
			DownloadTag: fmt.Sprintf(
				"%s [%s] (%s)",
				fp.SanitizePath(galleryResponse.GetTitle(), false),
				galleryLanguage,
				galleryResponse.GalleryID.String(),
			),
			FileName: fp.GetFileName(imageURL),
			FileURI:  imageURL,
		})
	}

	return galleryItems
}
