package deviantart

import (
	"path"
	"strconv"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules/deviantart/api"
)

func (m *deviantArt) parseGallery(item *models.TrackedItem) error {
	var downloadQueue []downloadQueueItem

	currentItemID, _ := strconv.ParseInt(item.CurrentItem, 10, 64)
	foundCurrentItem := false
	username := m.daPattern.galleryPattern.FindStringSubmatch(item.URI)[1]
	galleryID := m.daPattern.galleryPattern.FindStringSubmatch(item.URI)[2]
	galleryIntID, _ := strconv.ParseInt(galleryID, 10, 64)

	galleryUUID, err := m.daAPI.GalleryFolderIDToUUID(username, int(galleryIntID))
	if err != nil {
		return err
	}

	galleryName, err := m.daAPI.GalleryNameFromID(username, int(galleryIntID))
	if err != nil {
		return err
	}

	response, err := m.daAPI.Gallery(username, galleryUUID, 0, api.MaxDeviationsPerPage)
	if err != nil {
		return err
	}

	for !foundCurrentItem {
		for _, deviation := range response.Results {
			publishedTime, err := strconv.ParseInt(deviation.PublishedTime, 10, 64)
			if err != nil {
				return err
			}

			if item.CurrentItem == "" || publishedTime > currentItemID {
				downloadQueue = append(downloadQueue, downloadQueueItem{
					itemID:      deviation.PublishedTime,
					deviation:   deviation,
					downloadTag: path.Join(m.SanitizePath(username, false), galleryName),
				})
			} else {
				foundCurrentItem = true
				break
			}
		}

		if response.NextOffset == nil {
			break
		}

		response, err = m.daAPI.Gallery(username, galleryUUID, *response.NextOffset, api.MaxDeviationsPerPage)
		if err != nil {
			return err
		}
	}

	for i, j := 0, len(downloadQueue)-1; i < j; i, j = i+1, j-1 {
		downloadQueue[i], downloadQueue[j] = downloadQueue[j], downloadQueue[i]
	}

	return m.processDownloadQueue(downloadQueue, item)
}
