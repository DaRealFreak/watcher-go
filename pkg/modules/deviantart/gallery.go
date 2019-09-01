package deviantart

import (
	"fmt"

	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
)

func (m *deviantArt) parseGallery(item *models.TrackedItem) {
	userName := m.userGalleryPattern.FindStringSubmatch(item.URI)[1]

	foundCurrentItem := false
	offset := 0
	var deviations []*Deviation

	for !foundCurrentItem {
		results, apiErr := m.GalleryAll(userName, uint(offset), 24)
		if apiErr != nil {
			raven.CheckError(fmt.Errorf(apiErr.ErrorDescription))
		}

		for _, result := range results.Results {
			if result.DeviationID.String() == item.CurrentItem {
				foundCurrentItem = true
				break
			}
			deviations = append(deviations, result)
			fmt.Println(result.Title, result.DeviationID)
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

	fmt.Println(m.retrieveDeviationDetails(deviations))
}
