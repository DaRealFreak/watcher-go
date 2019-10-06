package deviantart

import (
	"fmt"
	"strings"

	"github.com/DaRealFreak/watcher-go/pkg/models"
)

func (m *deviantArt) parseCollection(appURL string, item *models.TrackedItem) error {
	userName := strings.Split(appURL, "/")[3]
	folderID := strings.Split(appURL, "/")[4]
	foundCurrentItem := false
	offset := 0
	var deviations []*Deviation

	for !foundCurrentItem {
		results, apiErr, err := m.Collections(userName, folderID, uint(offset), 24)
		if err != nil {
			return err
		}
		if apiErr != nil {
			return fmt.Errorf(apiErr.ErrorDescription)
		}

		for _, result := range results.Results {
			if result.DeviationID.String() == item.CurrentItem {
				foundCurrentItem = true
				break
			}
			// fake the username to retrieve owner/folderID as path
			result.Author.Username = userName + "/" + folderID
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
