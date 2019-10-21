package deviantart

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/DaRealFreak/watcher-go/pkg/models"
)

func (m *deviantArt) parseGalleryAll(item *models.TrackedItem) error {
	userName := m.userGalleryPattern.FindStringSubmatch(item.URI)[1]

	foundCurrentItem := false
	offset := 0
	var deviations []*Deviation

	for !foundCurrentItem {
		results, apiErr, err := m.GalleryAll(userName, uint(offset), 24)
		if err != nil {
			return err
		}
		if apiErr != nil {
			return fmt.Errorf(apiErr.ErrorDescription)
		}

		for _, result := range results.Results {
			// will return 0 on error, so fine for us too
			currentItemID, _ := strconv.ParseInt(item.CurrentItem, 10, 64)
			itemID, _ := strconv.ParseInt(result.PublishedTime, 10, 64)

			if !(item.CurrentItem == "" || itemID > currentItemID) {
				foundCurrentItem = true
				break
			}

			deviations = append(deviations, result)
		}

		// no more results, break out of the loop
		if !results.HasMore {
			break
		}

		// update offset
		nextOffset, err := results.NextOffset.Int64()
		if err != nil {
			return err
		}
		offset = int(nextOffset)
	}

	// reverse deviations to download the oldest items first
	for i, j := 0, len(deviations)-1; i < j; i, j = i+1, j-1 {
		deviations[i], deviations[j] = deviations[j], deviations[i]
	}
	// retrieve all relevant details and parse the download queue
	err := m.processDownloadQueue(item, deviations)
	if err == nil {
		results, _, err := m.GalleryAll(userName, uint(offset), 24)
		if err == nil && results != nil {
			m.DbIO.UpdateTrackedItem(item, results.Results[0].PublishedTime)
		}
	}
	return err
}

func (m *deviantArt) parseGallery(appURL string, item *models.TrackedItem) error {
	userName := strings.Split(appURL, "/")[3]
	folderID := strings.Split(appURL, "/")[4]
	foundCurrentItem := false
	offset := 0
	var deviations []*Deviation

	for !foundCurrentItem {
		results, apiErr, err := m.Gallery(userName, folderID, "newest", uint(offset), 24)
		if err != nil {
			return err
		}
		if apiErr != nil {
			return fmt.Errorf(apiErr.ErrorDescription)
		}

		for _, result := range results.Results {
			// will return 0 on error, so fine for us too
			currentItemID, _ := strconv.ParseInt(item.CurrentItem, 10, 64)
			itemID, _ := strconv.ParseInt(result.PublishedTime, 10, 64)

			if !(item.CurrentItem == "" || itemID > currentItemID) {
				foundCurrentItem = true
				break
			}

			deviations = append(deviations, result)
		}

		// no more results, break out of the loop
		if !results.HasMore {
			break
		}

		// update offset
		nextOffset, err := results.NextOffset.Int64()
		if err != nil {
			return err
		}
		offset = int(nextOffset)
	}

	// reverse deviations to download the oldest items first
	for i, j := 0, len(deviations)-1; i < j; i, j = i+1, j-1 {
		deviations[i], deviations[j] = deviations[j], deviations[i]
	}
	// retrieve all relevant details and parse the download queue
	return m.processDownloadQueue(item, deviations)
}
