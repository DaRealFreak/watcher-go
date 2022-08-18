package deviantart

import (
	"strconv"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules/deviantart/api"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
)

func (m *deviantArt) parseTagDevAPI(item *models.TrackedItem) error {
	var downloadQueue []downloadQueueItemDevAPI

	tag := m.daPattern.tagPattern.FindStringSubmatch(item.URI)[1]
	currentItemID, _ := strconv.ParseInt(item.CurrentItem, 10, 64)
	foundCurrentItem := false

	response, err := m.daAPI.BrowseTags(tag, 0, api.MaxDeviationsPerPage)
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
					downloadTag: fp.SanitizePath(tag, false),
				})
			} else {
				foundCurrentItem = true
				break
			}
		}

		if response.NextOffset == nil || foundCurrentItem {
			break
		}

		response, err = m.daAPI.BrowseTags(tag, *response.NextOffset, api.MaxDeviationsPerPage)
		if err != nil {
			return err
		}
	}

	for i, j := 0, len(downloadQueue)-1; i < j; i, j = i+1, j-1 {
		downloadQueue[i], downloadQueue[j] = downloadQueue[j], downloadQueue[i]
	}

	return m.processDownloadQueue(downloadQueue, item)
}
