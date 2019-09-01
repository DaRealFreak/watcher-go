package deviantart

import (
	"fmt"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
	"github.com/jaytaylor/html2text"
	"github.com/spf13/viper"
	"io/ioutil"
	"os"
	"path"
)

// DeviationItem contains the combination of the HTML content and Download information
type DeviationItem struct {
	*Deviation
	*DeviationContent
	Download *Image
}

// retrieveDeviationDetails adds possible Content or Download responses if required
func (m *deviantArt) retrieveDeviationDetails(deviation *Deviation) (completedDeviationItem *DeviationItem) {
	completedDeviationItem = &DeviationItem{
		Deviation: deviation,
	}
	if deviation.Excerpt != "" {
		// deviation has text so we retrieve the full content
		deviationContent, apiErr := m.DeviationContent(deviation.DeviationID.String())
		if apiErr != nil {
			raven.CheckError(fmt.Errorf(apiErr.ErrorDescription))
		}
		completedDeviationItem.DeviationContent = deviationContent
	}
	if deviation.IsDownloadable {
		deviationDownload, apiErr := m.DeviationDownload(deviation.DeviationID.String())
		if apiErr != nil {
			raven.CheckError(fmt.Errorf(apiErr.ErrorDescription))
		}
		completedDeviationItem.Download = deviationDownload
	}
	return completedDeviationItem
}

func (m *deviantArt) processDownloadQueue(trackedItem *models.TrackedItem, deviations []*Deviation) {
	for _, item := range deviations {
		deviationItem := m.retrieveDeviationDetails(item)

		// ensure download directory, needed for only text artists
		m.Session.EnsureDownloadDirectory(
			path.Join(
				viper.GetString("download.directory"),
				m.Key(),
				deviationItem.Author.Username,
				"tmp.txt",
			),
		)
		if deviationItem.Download != nil {
			raven.CheckError(m.Session.DownloadFile(
				path.Join(viper.GetString("download.directory"),
					m.Key(),
					deviationItem.Author.Username,
					deviationItem.PublishedTime+"_"+m.GetFileName(deviationItem.Download.Src),
				),
				deviationItem.Download.Src,
			))
		}
		if deviationItem.DeviationContent != nil {
			text, err := html2text.FromString(deviationItem.DeviationContent.HTML)
			raven.CheckError(err)

			filePath := path.Join(viper.GetString("download.directory"),
				m.Key(),
				deviationItem.Author.Username,
				deviationItem.PublishedTime+"_"+m.SanitizePath(deviationItem.Title, false)+".txt",
			)
			err = ioutil.WriteFile(filePath, []byte(text), os.ModePerm)
			raven.CheckError(err)
		}
		if deviationItem.Content != nil {
			raven.CheckError(m.Session.DownloadFile(
				path.Join(viper.GetString("download.directory"),
					m.Key(),
					deviationItem.Author.Username,
					deviationItem.PublishedTime+"_"+m.GetFileName(deviationItem.Content.Src),
				),
				deviationItem.Content.Src,
			))
		}
		if deviationItem.Content == nil && len(deviationItem.Thumbs) > 0 {
			// if no content is set we download the highest thumbnail
			last := deviationItem.Thumbs[len(deviationItem.Thumbs)-1]
			raven.CheckError(m.Session.DownloadFile(
				path.Join(viper.GetString("download.directory"),
					m.Key(),
					deviationItem.Author.Username,
					deviationItem.PublishedTime+"_"+m.GetFileName(last.Src),
				),
				last.Src,
			))
		}
		m.DbIO.UpdateTrackedItem(trackedItem, deviationItem.DeviationID.String())
	}
}
