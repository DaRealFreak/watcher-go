package chounyuu

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules/chounyuu/api"
)

// parseThread parses thread searches
func (m *chounyuu) parseThread(item *models.TrackedItem) error {
	pageDomain := api.ChounyuuDomain
	if strings.Contains(item.URI, api.SuperFutaDomain) {
		pageDomain = api.SuperFutaDomain
	}

	threadPattern := regexp.MustCompile(`/thread/(?P<ThreadID>.*)/`)
	if threadPattern.MatchString(item.URI) {
		threadID := threadPattern.FindStringSubmatch(item.URI)[1]
		collection, err := m.api.RetrieveThreadResponse(pageDomain, threadID, 1)
		if err != nil {
			return err
		}

		lastPage := collection.Thread.Pages

		var downloadQueue []models.DownloadQueueItem

		currentItemID, _ := strconv.ParseInt(item.CurrentItem, 10, 64)
		foundCurrentItem := false

		for !foundCurrentItem {
			collection, err = m.api.RetrieveThreadResponse(pageDomain, threadID, lastPage)
			if err != nil {
				return err
			}

			for i := len(collection.Thread.Items.Posts) - 1; i >= 0; i = i - 1 {
				threadItem := collection.Thread.Items.Posts[i]

				if int(currentItemID) >= threadItem.ID {
					foundCurrentItem = true
					break
				}

				if threadItem.ImageID > 0 {
					downloadQueue = append(downloadQueue, models.DownloadQueueItem{
						ItemID:      strconv.Itoa(threadItem.ID),
						DownloadTag: fmt.Sprintf("%d_%s", collection.Thread.Items.ID, collection.Thread.Items.Title),
						FileURI:     fmt.Sprintf("https://images.%s/src/%s", api.ChounyuuDomain, threadItem.Filename),
						FileName:    fmt.Sprintf("%d_%s", threadItem.ImageID, threadItem.Filename),
					})
				}
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

		if err = m.processDownloadQueue(downloadQueue, item); err != nil {
			return err
		}

		// set item status to complete if the thread is closed
		if collection.Thread.Items.Closed {
			m.DbIO.ChangeTrackedItemCompleteStatus(item, true)
		}
	}

	return nil
}
