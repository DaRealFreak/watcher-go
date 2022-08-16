package deviantart

import (
	"strconv"
	"time"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules/deviantart/napi"
)

func (m *deviantArt) parseTagNapi(item *models.TrackedItem) error {
	var downloadQueue []downloadQueueItemNAPI

	tag := m.daPattern.tagPattern.FindStringSubmatch(item.URI)[1]
	currentItemID, _ := strconv.ParseInt(item.CurrentItem, 10, 64)
	foundCurrentItem := false

	response, err := m.nAPI.DeviationTag(tag, "", napi.OrderMostRecent)
	if err != nil {
		return err
	}

	for !foundCurrentItem {
		for _, deviation := range response.Deviations {
			if deviation.Type == "tier" {
				// tier entries do not respect the "most-recent" order and have no content most of the time
				continue
			}

			t, dateErr := time.Parse(napi.DateLayout, deviation.PublishedTime)
			if dateErr != nil {
				return dateErr
			}

			if item.CurrentItem == "" || t.Unix() > currentItemID {
				downloadQueue = append(downloadQueue, downloadQueueItemNAPI{
					itemID:      strconv.Itoa(int(t.Unix())),
					deviation:   deviation,
					downloadTag: m.SanitizePath(tag, false),
				})
			} else {
				foundCurrentItem = true
				break
			}
		}

		if !response.HasMore || foundCurrentItem {
			break
		}

		response, err = m.nAPI.DeviationTag(tag, response.NextCursor, napi.OrderMostRecent)
		if err != nil {
			return err
		}
	}

	for i, j := 0, len(downloadQueue)-1; i < j; i, j = i+1, j-1 {
		downloadQueue[i], downloadQueue[j] = downloadQueue[j], downloadQueue[i]
	}

	return m.processDownloadQueueNapi(downloadQueue, item)
}
