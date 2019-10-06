package deviantart

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/DaRealFreak/watcher-go/pkg/models"
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
func (m *deviantArt) parseDeviation(appURL string, item *models.TrackedItem) error {
	deviationID := strings.Split(appURL, "/")[3]
	result, apiErr, err := m.Deviation(deviationID)
	if err != nil {
		return nil
	}
	if apiErr != nil {
		return fmt.Errorf(apiErr.ErrorDescription)
	}
	if err := m.processDownloadQueue(item, []*Deviation{result}); err != nil {
		return err
	}
	m.DbIO.ChangeTrackedItemCompleteStatus(item, true)
	return nil
}

// retrieveDeviationDetails adds possible Content or Download responses if required
func (m *deviantArt) retrieveDeviationDetails(deviation *Deviation) (completedDeviationItem *DeviationItem, err error) {
	completedDeviationItem = &DeviationItem{
		Deviation: deviation,
	}
	if deviation.Excerpt != "" {
		// deviation has text so we retrieve the full content
		deviationContent, apiErr, err := m.DeviationContent(deviation.DeviationID.String())
		if err != nil {
			return nil, err
		}
		if apiErr != nil {
			return nil, fmt.Errorf(apiErr.ErrorDescription)
		}
		completedDeviationItem.DeviationContent = deviationContent
	}
	if deviation.IsDownloadable {
		deviationDownload, err := m.DeviationDownloadFallback(deviation.URL)
		if err != nil {
			var apiErr *APIError
			deviationDownload, apiErr, err = m.DeviationDownload(deviation.DeviationID.String())
			if err != nil {
				return nil, err
			}
			if apiErr != nil {
				return nil, fmt.Errorf(apiErr.ErrorDescription)
			}
		}
		completedDeviationItem.Download = deviationDownload
	}
	return completedDeviationItem, nil
}

// processDownloadQueue retrieves the deviation details and proceeds to download the relevant information
func (m *deviantArt) processDownloadQueue(trackedItem *models.TrackedItem, deviations []*Deviation) error {
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

		deviationItem, err := m.retrieveDeviationDetails(item)
		if err != nil {
			return err
		}

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
			if err := m.Session.DownloadFile(
				path.Join(viper.GetString("download.directory"),
					m.Key(),
					deviationItem.Author.Username,
					deviationItem.PublishedTime+"_"+m.GetFileName(deviationItem.Download.Src),
				),
				deviationItem.Download.Src,
			); err != nil {
				return err
			}
		}
		if deviationItem.DeviationContent != nil {
			text, err := html2text.FromString(deviationItem.DeviationContent.HTML)
			if err != nil {
				return err
			}

			filePath := path.Join(viper.GetString("download.directory"),
				m.Key(),
				deviationItem.Author.Username,
				deviationItem.PublishedTime+"_"+m.SanitizePath(deviationItem.Title, false)+".txt",
			)
			if err := ioutil.WriteFile(filePath, []byte(text), os.ModePerm); err != nil {
				return err
			}
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
				if err := m.Session.DownloadFile(
					path.Join(viper.GetString("download.directory"),
						m.Key(),
						deviationItem.Author.Username,
						deviationItem.PublishedTime+"_"+m.GetFileName(deviationItem.Content.Src),
					),
					deviationItem.Content.Src,
				); err != nil {
					return err
				}
			case len(deviationItem.Thumbs) > 0:
				// if no content is set we download the highest thumbnail
				last := deviationItem.Thumbs[len(deviationItem.Thumbs)-1]
				if err := m.Session.DownloadFile(
					path.Join(viper.GetString("download.directory"),
						m.Key(),
						deviationItem.Author.Username,
						deviationItem.PublishedTime+"_"+m.GetFileName(last.Src),
					),
					last.Src,
				); err != nil {
					return err
				}
			}
		}

		m.DbIO.UpdateTrackedItem(trackedItem, deviationItem.DeviationID.String())
	}
	return nil
}
