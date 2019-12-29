package deviantart

import (
	"fmt"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/DaRealFreak/watcher-go/pkg/modules/deviantart/api"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"path"
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

	}
	return nil
}
