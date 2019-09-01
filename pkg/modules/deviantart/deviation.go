package deviantart

import (
	"fmt"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
)

// DeviationItem contains the combination of the HTML content and Download information
type DeviationItem struct {
	*Deviation
	*DeviationContent
	Download *Image
}

// retrieveDeviationDetails adds possible Content or Download responses if required
func (m *deviantArt) retrieveDeviationDetails(deviations []*Deviation) (completedDeviationItems []*DeviationItem) {
	for _, result := range deviations {
		completedDeviationItem := &DeviationItem{
			Deviation: result,
		}
		if result.Excerpt != "" {
			// deviation has text so we retrieve the full content
			deviationContent, apiErr := m.DeviationContent(result.DeviationID.String())
			if apiErr != nil {
				raven.CheckError(fmt.Errorf(apiErr.ErrorDescription))
			}
			completedDeviationItem.DeviationContent = deviationContent
		}
		if result.IsDownloadable {
			deviationDownload, apiErr := m.DeviationDownload(result.DeviationID.String())
			if apiErr != nil {
				raven.CheckError(fmt.Errorf(apiErr.ErrorDescription))
			}
			completedDeviationItem.Download = deviationDownload
		}
		completedDeviationItems = append(completedDeviationItems, completedDeviationItem)
	}
	return completedDeviationItems
}
