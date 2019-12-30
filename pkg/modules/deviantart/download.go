package deviantart

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/DaRealFreak/watcher-go/pkg/modules/deviantart/api"
	"github.com/jaytaylor/html2text"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type downloadQueueItem struct {
	itemID      string
	deviation   api.Deviation
	downloadTag string
}

func (m *deviantArt) processDownloadQueue(downloadQueue []downloadQueueItem, trackedItem *models.TrackedItem) error {
	log.WithField("module", m.Key).Info(
		fmt.Sprintf("found %d new items for uri: %s", len(downloadQueue), trackedItem.URI),
	)

	for index, deviationItem := range downloadQueue {
		log.WithField("module", m.Key).Info(
			fmt.Sprintf(
				"downloading updates for uri: %s (%0.2f%%)",
				trackedItem.URI,
				float64(index+1)/float64(len(downloadQueue))*100,
			),
		)

		// ensure download directory, needed for only text artists
		m.Session.EnsureDownloadDirectory(
			path.Join(
				viper.GetString("download.directory"),
				m.Key,
				deviationItem.deviation.Author.Username,
				"tmp.txt",
			),
		)

		if deviationItem.deviation.Excerpt != nil {
			if err := m.downloadHTMLContent(deviationItem.deviation); err != nil {
				return err
			}
		}

		if deviationItem.deviation.IsDownloadable {
			if err := m.downloadDeviation(deviationItem.deviation); err != nil {
				return err
			}
		}

		m.DbIO.UpdateTrackedItem(trackedItem, deviationItem.deviation.PublishedTime)
	}

	return nil
}

func (m *deviantArt) downloadDeviation(deviation api.Deviation) error {
	deviationDownload, err := m.daAPI.DeviationDownloadFallback(deviation.DeviationURL)
	if err != nil {
		deviationDownload, err = m.daAPI.DeviationDownload(deviation.DeviationID)
		if err != nil {
			return err
		}
	}

	if err := m.Session.DownloadFile(
		path.Join(viper.GetString("download.directory"),
			m.Key,
			deviation.Author.Username,
			deviation.PublishedTime+"_"+m.GetFileName(deviationDownload.Src),
		),
		deviationDownload.Src,
	); err != nil {
		return err
	}

	return nil
}

func (m *deviantArt) downloadHTMLContent(deviation api.Deviation) error {
	// deviation has text so we retrieve the full content
	deviationContent, err := m.daAPI.DeviationContent(deviation.DeviationID)
	if err != nil {
		return err
	}

	text, err := html2text.FromString(deviationContent.HTML)
	if err != nil {
		return err
	}

	filePath := path.Join(viper.GetString("download.directory"),
		m.Key,
		deviation.Author.Username,
		fmt.Sprintf(
			"%s_%s.txt",
			deviation.PublishedTime,
			m.SanitizePath(deviation.Title, false),
		),
	)
	if err := ioutil.WriteFile(filePath, []byte(text), os.ModePerm); err != nil {
		return err
	}

	return nil
}
