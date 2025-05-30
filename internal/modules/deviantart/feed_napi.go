package deviantart

import (
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
	"strconv"
)

func (m *deviantArt) parseFeedNapi(item *models.TrackedItem) error {
	var downloadQueue []downloadQueueItemNAPI

	currentItemID, _ := strconv.ParseInt(item.CurrentItem, 10, 64)
	foundCurrentItem := false

	response, err := m.nAPI.DeviationsFeed("")
	if err != nil {
		return err
	}

	if m.settings.MultiProxy {
		raven.CheckError(m.setProxyMethod())
	}

	if item.SubFolder == "" {
		m.DbIO.ChangeTrackedItemSubFolder(item, "home_feed")
	}

	for {
		for _, deviation := range response.Deviations {
			if deviation.Type == "tier" {
				// tier entries do not respect the "most-recent" order and have no content most of the time
				continue
			}

			if item.CurrentItem == "" || deviation.GetPublishedTime().Unix() > currentItemID {
				downloadQueue = append(downloadQueue, downloadQueueItemNAPI{
					itemID:      deviation.GetPublishedTimestamp(),
					deviation:   deviation,
					downloadTag: fp.SanitizePath(item.SubFolder, false),
				})
			} else {
				foundCurrentItem = true
				break
			}
		}

		if !response.HasMore || foundCurrentItem {
			break
		}

		response, err = m.nAPI.DeviationsFeed(response.NextCursor)
		if err != nil {
			return err
		}

		if m.settings.MultiProxy {
			raven.CheckError(m.setProxyMethod())
		}
	}

	for i, j := 0, len(downloadQueue)-1; i < j; i, j = i+1, j-1 {
		downloadQueue[i], downloadQueue[j] = downloadQueue[j], downloadQueue[i]
	}

	return m.processDownloadQueueNapi(downloadQueue, item)
}
