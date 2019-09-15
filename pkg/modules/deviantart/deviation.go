package deviantart

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
	"github.com/jaytaylor/html2text"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// DeviationItem contains the combination of the HTML content and Download information
type DeviationItem struct {
	*Deviation
	*DeviationContent
	Download *Image
}

// parseDeviation parses and downloads a single deviation
func (m *deviantArt) parseDeviation(appURL string, item *models.TrackedItem) {
	deviationID := strings.Split(appURL, "/")[3]
	result, apiErr := m.Deviation(deviationID)
	if apiErr != nil {
		raven.CheckError(fmt.Errorf(apiErr.ErrorDescription))
	}
	m.processDownloadQueue(item, []*Deviation{result})
	m.DbIO.ChangeTrackedItemCompleteStatus(item, true)
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
		deviationDownload, err := m.DeviationDownloadFallback(deviation.URL)
		if err != nil {
			var apiErr *APIError
			deviationDownload, apiErr = m.DeviationDownload(deviation.DeviationID.String())
			if apiErr != nil {
				raven.CheckError(fmt.Errorf(apiErr.ErrorDescription))
			}
		}
		completedDeviationItem.Download = deviationDownload
	}
	return completedDeviationItem
}

// processDownloadQueue retrieves the deviation details and proceeds to download the relevant information
func (m *deviantArt) processDownloadQueue(trackedItem *models.TrackedItem, deviations []*Deviation) {
	log.WithField("module", m.Key()).Info(
		fmt.Sprintf("found %d new items for uri: %s", len(deviations), trackedItem.URI),
	)

	for index, item := range deviations {
		log.WithField("module", m.Key()).Info(
			fmt.Sprintf(
				"downloading updates for uri: %s (%0.2f%%)",
				trackedItem.URI,
				float64(index+1)/float64(len(deviations))*100,
			),
		)

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

		// download if one of these conditions match:
		// - no download link (content)
		// - HTML deviation (content or thumbnail)
		// - download link and content but different file types (f.e. image + pdf)
		if deviationItem.Download == nil ||
			deviationItem.DeviationContent != nil ||
			(deviationItem.Download != nil && deviationItem.Content != nil &&
				(m.GetFileExtension(deviationItem.Download.Src) != m.GetFileExtension(deviationItem.Content.Src))) {
			// if we have an HTML story here we are downloading the content/thumbs too
			switch {
			case deviationItem.Content != nil:
				raven.CheckError(m.Session.DownloadFile(
					path.Join(viper.GetString("download.directory"),
						m.Key(),
						deviationItem.Author.Username,
						deviationItem.PublishedTime+"_"+m.GetFileName(deviationItem.Content.Src),
					),
					deviationItem.Content.Src,
				))
			case len(deviationItem.Thumbs) > 0:
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
		}

		m.DbIO.UpdateTrackedItem(trackedItem, deviationItem.DeviationID.String())
	}
}
