package deviantart

import (
	"fmt"

	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
)

// parseFeed retrieves the deviation_submitted bucket and parses all new deviations
func (m *deviantArt) parseFeed(item *models.TrackedItem) {
	foundCurrentItem := false
	var deviations []*Deviation

	bucket, apiErr := m.FeedHomeBucket("deviation_submitted", 0, 10, true)
	if apiErr != nil {
		raven.CheckError(fmt.Errorf(apiErr.ErrorDescription))
	}
	for !foundCurrentItem {
		results, apiErr := m.FeedHome(bucket.Cursor, true)
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

		// update cursor to retrieve next page
		bucket.Cursor = results.Cursor
	}

	// reverse deviations to download the oldest items first
	for i, j := 0, len(deviations)-1; i < j; i, j = i+1, j-1 {
		deviations[i], deviations[j] = deviations[j], deviations[i]
	}
	// retrieve all relevant details and parse the download queue
	m.processDownloadQueue(item, deviations)
}
