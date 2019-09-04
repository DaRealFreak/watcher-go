package deviantart

import (
	"fmt"
	"strings"

	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
)

func (m *deviantArt) parseTag(appURL string, item *models.TrackedItem) {
	tag := strings.Split(appURL, "/")[3]
	foundCurrentItem := false
	offset := 0
	var deviations []*Deviation

	for !foundCurrentItem {
		results, apiErr := m.BrowseTags(tag, uint(offset), 24)
		if apiErr != nil {
			raven.CheckError(fmt.Errorf(apiErr.ErrorDescription))
		}

		for _, result := range results.Results {
			if result.DeviationID.String() == item.CurrentItem {
				foundCurrentItem = true
				break
			}
			// fake the username to save in a specific directory
			// ToDo: use custom result struct
			result.Author.Username = tag
			deviations = append(deviations, result)
		}

		// no more results, break out of the loop
		if !results.HasMore {
			break
		}

		// update offset
		nextOffset, err := results.NextOffset.Int64()
		raven.CheckError(err)
		offset = int(nextOffset)
	}

	// reverse deviations to download the oldest items first
	for i, j := 0, len(deviations)-1; i < j; i, j = i+1, j-1 {
		deviations[i], deviations[j] = deviations[j], deviations[i]
	}
	// retrieve all relevant details and parse the download queue
	m.processDownloadQueue(item, deviations)
}
