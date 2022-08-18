package deviantart

import (
	"path"
	"strconv"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules/deviantart/api"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
)

func (m *deviantArt) parseGalleryDevAPI(item *models.TrackedItem) error {
	var downloadQueue []downloadQueueItemDevAPI

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
			var publishedTime int64
			publishedTime, err = strconv.ParseInt(deviation.PublishedTime, 10, 64)
			if err != nil {
				return err
			}

			if item.CurrentItem == "" || publishedTime > currentItemID {
				downloadQueue = append(downloadQueue, downloadQueueItemDevAPI{
					itemID:      deviation.PublishedTime,
					deviation:   deviation,
					downloadTag: path.Join(fp.SanitizePath(username, false), fp.SanitizePath(galleryName, false)),
				})
			} else {
				foundCurrentItem = true
				break
			}
		}

		if response.NextOffset == nil || foundCurrentItem {
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
