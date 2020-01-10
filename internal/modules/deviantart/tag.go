// Package deviantart contains the implementation of the deviantart module
// nolint: dupl
package deviantart

import (
	"strconv"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules/deviantart/api"
)

func (m *deviantArt) parseTag(item *models.TrackedItem) error {
	var downloadQueue []downloadQueueItem

	tag := m.daPattern.tagPattern.FindStringSubmatch(item.URI)[1]
	currentItemID, _ := strconv.ParseInt(item.CurrentItem, 10, 64)
	foundCurrentItem := false

	response, err := m.daAPI.BrowseTags(tag, 0, api.MaxDeviationsPerPage)
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
					downloadTag: m.SanitizePath(tag, false),
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
