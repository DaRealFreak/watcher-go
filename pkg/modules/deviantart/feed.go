package deviantart

import (
	"fmt"

	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
)

func (m *deviantArt) parseFeed(item *models.TrackedItem) {
	foundCurrentItem := false
	offset := 0
	var deviations []*Deviation

	for !foundCurrentItem {
		results, apiErr := m.FeedHomeBucket("deviation_submitted", uint(offset), 10, true)
		if apiErr != nil {
			raven.CheckError(fmt.Errorf(apiErr.ErrorDescription))
		}

		for _, itemFeed := range results.Items {
			for _, result := range itemFeed.Deviations {
				if result.DeviationID.String() == item.CurrentItem {
					foundCurrentItem = true
					break
				}
				deviations = append(deviations, result)
			}
			// break outer loop too if current item got found
			if foundCurrentItem {
				break
			}
		}

		// no more results, break out of the loop
		if !results.HasMore {
			break
		}

		// update offset
		// feed API documentation is completely wrong (response and parameters), ignores limit anyways
		offset += 10
	}

	// reverse deviations to download the oldest items first
	for i, j := 0, len(deviations)-1; i < j; i, j = i+1, j-1 {
		deviations[i], deviations[j] = deviations[j], deviations[i]
	}
	// retrieve all relevant details and parse the download queue
	m.processDownloadQueue(item, deviations)
}
