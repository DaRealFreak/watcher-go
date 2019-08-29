package deviantart

import (
	"fmt"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
)

func (m *deviantArt) parseGallery(item *models.TrackedItem) {
	userName := m.userGalleryPattern.FindStringSubmatch(item.URI)[1]
	results, apiErr := m.GalleryAll(userName, 0, 24)
	deviations := results.Results
	for results.HasMore {
		nextOffset, err := results.NextOffset.Int64()
		raven.CheckError(err)

		results, apiErr = m.GalleryAll(userName, uint(nextOffset), 24)
		if apiErr != nil {
			raven.CheckError(fmt.Errorf(apiErr.ErrorDescription))
		}
		deviations = append(deviations, results.Results...)
	}

	count := 0
	count2 := 0
	for _, result := range deviations {
		if result.Excerpt != "" {
			count++
		}
		content, _ := m.DeviationContent(result.DeviationID.String())
		if content != nil {
			count2++
		}
	}
	fmt.Println(count, count2)
}
