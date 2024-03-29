package deviantart

import (
	"github.com/DaRealFreak/watcher-go/pkg/fp"
	"strconv"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules/deviantart/api"
)

func (m *deviantArt) parseFeedDevApi(item *models.TrackedItem) error {
	var downloadQueue []downloadQueueItemDevAPI

	currentItemID, _ := strconv.ParseInt(item.CurrentItem, 10, 64)
	foundCurrentItem := false

	response, err := m.daAPI.FeedHomeBucket(api.BucketDeviationSubmitted, 0)
	if err != nil {
		return err
	}

	if item.SubFolder == "" {
		m.DbIO.ChangeTrackedItemSubFolder(item, "home_feed")
	}

	for !foundCurrentItem {
		for _, feedItem := range response.Items {
			var publishedTime int64
			publishedTime, err = strconv.ParseInt(feedItem.Timestamp, 10, 64)
			if err != nil {
				return err
			}

			if item.CurrentItem == "" || publishedTime > currentItemID {
				for _, singleDeviation := range feedItem.Deviations {
					downloadQueue = append(downloadQueue, downloadQueueItemDevAPI{
						itemID:      singleDeviation.PublishedTime,
						deviation:   singleDeviation,
						downloadTag: fp.SanitizePath(item.SubFolder, false),
					})
				}
			} else {
				foundCurrentItem = true
				break
			}
		}

		if !response.HasMore {
			break
		}

		response, err = m.daAPI.FeedHome(response.Cursor)
		if err != nil {
			return err
		}
	}

	for i, j := 0, len(downloadQueue)-1; i < j; i, j = i+1, j-1 {
		downloadQueue[i], downloadQueue[j] = downloadQueue[j], downloadQueue[i]
	}

	return m.processDownloadQueue(downloadQueue, item)
}
