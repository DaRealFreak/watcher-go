package chounyuu

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/modules/chounyuu/api"

	"github.com/DaRealFreak/watcher-go/internal/models"
)

// parseTag parses tag searches
func (m *chounyuu) parseTag(item *models.TrackedItem) error {
	pageDomain := api.ChounyuuDomain
	if strings.Contains(item.URI, api.SuperFutaDomain) {
		pageDomain = api.SuperFutaDomain
	}

	tagPattern := regexp.MustCompile(`/tag/(?P<TagID>.*)/`)
	if tagPattern.MatchString(item.URI) {
		tagId := tagPattern.FindStringSubmatch(item.URI)[1]
		collection, err := m.api.RetrieveTagResponse(pageDomain, tagId, 1)
		if err != nil {
			return err
		}

		lastPage := collection.Tag.Pages

		var downloadQueue []models.DownloadQueueItem

		currentItemID, _ := strconv.ParseInt(item.CurrentItem, 10, 64)
		foundCurrentItem := false

		for !foundCurrentItem {
			collection, err = m.api.RetrieveTagResponse(pageDomain, tagId, lastPage)
			if err != nil {
				return err
			}

			for i := len(collection.Tag.Items.Images) - 1; i >= 0; i = i - 1 {
				tagItem := collection.Tag.Items.Images[i]

				if int(currentItemID) >= tagItem.ID {
					foundCurrentItem = true
					break
				}

				downloadQueue = append(downloadQueue, models.DownloadQueueItem{
					ItemID:      strconv.Itoa(tagItem.ID),
					DownloadTag: collection.Tag.Items.Tag,
					FileURI:     fmt.Sprintf("https://images.%s/src/%s", api.ChounyuuDomain, tagItem.Filename),
					FileName:    fmt.Sprintf("%d_%s", tagItem.ID, tagItem.Filename),
				})
			}

			if lastPage == 1 {
				break
			} else {
				lastPage = lastPage - 1
			}
		}

		// reverse download queue to download old items first
		for i, j := 0, len(downloadQueue)-1; i < j; i, j = i+1, j-1 {
			downloadQueue[i], downloadQueue[j] = downloadQueue[j], downloadQueue[i]
		}

		return m.ProcessDownloadQueue(downloadQueue, item)
	} else {
		return nil
	}
}
