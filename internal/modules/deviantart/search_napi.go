package deviantart

import (
	"net/url"
	"strconv"
	"time"

	"github.com/DaRealFreak/watcher-go/internal/modules/deviantart/napi"

	"github.com/DaRealFreak/watcher-go/internal/models"
)

func (m *deviantArt) parseSearchNapi(item *models.TrackedItem) error {
	var downloadQueue []downloadQueueItemNAPI

	u, err := url.Parse(item.URI)
	if err != nil {
		return err
	}

	parsedQueryString, _ := url.ParseQuery(u.RawQuery)
	searchQuery, _ := parsedQueryString["q"]
	searchTag := searchQuery[0]

	currentItemID, _ := strconv.ParseInt(item.CurrentItem, 10, 64)
	foundCurrentItem := false

	response, err := m.nAPI.DeviationSearch(searchTag, "", napi.OrderMostRecent)
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
					downloadTag: m.SanitizePath(searchTag, false),
				})
			} else {
				foundCurrentItem = true
				break
			}
		}

		if !response.HasMore || foundCurrentItem {
			break
		}

		response, err = m.nAPI.DeviationSearch(searchTag, response.NextCursor, napi.OrderMostRecent)
		if err != nil {
			return err
		}
	}

	for i, j := 0, len(downloadQueue)-1; i < j; i, j = i+1, j-1 {
		downloadQueue[i], downloadQueue[j] = downloadQueue[j], downloadQueue[i]
	}

	return m.processDownloadQueueNapi(downloadQueue, item)
}
