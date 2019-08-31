package deviantart

import (
	"fmt"

	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
)

func (m *deviantArt) parseGallery(item *models.TrackedItem) {
	userName := m.userGalleryPattern.FindStringSubmatch(item.URI)[1]
	results, apiErr := m.GalleryAll(userName, 0, 24)
	if apiErr != nil {
		raven.CheckError(fmt.Errorf(apiErr.ErrorDescription))
	}

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

	for _, result := range deviations {
		if result.Excerpt != "" {
			// deviation has text so we retrieve the full content
			_, _ = m.DeviationContent(result.DeviationID.String())
		}
	}
}