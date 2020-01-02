package twitter

import (
	"fmt"
	"path"
	"strconv"

	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/DaRealFreak/watcher-go/pkg/modules/twitter/api"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// processDownloadQueue downloads all media entities from the passed tweets if set
func (m *twitter) processDownloadQueue(downloadQueue []*api.Tweet, trackedItem *models.TrackedItem) error {
	log.WithField("module", m.Key).Info(
		fmt.Sprintf("found %d new items for uri: \"%s\"", len(downloadQueue), trackedItem.URI),
	)

	screenName, err := m.extractScreenName(trackedItem.URI)
	if err != nil {
		return err
	}

	for index, tweet := range downloadQueue {
		log.WithField("module", m.Key).Info(
			fmt.Sprintf(
				"downloading updates for uri: \"%s\" (%0.2f%%)",
				trackedItem.URI,
				float64(index+1)/float64(len(downloadQueue))*100,
			),
		)

		for _, entity := range tweet.ExtendedEntities.Media {
			if err := m.twitterAPI.Session.DownloadFile(
				path.Join(
					viper.GetString("download.directory"),
					m.Key,
					screenName,
					fmt.Sprintf("%d_%s", tweet.ID, m.GetFileName(entity.MediaURLHTTPS)),
				),
				entity.MediaURLHTTPS,
			); err != nil {
				return err
			}
		}

		m.DbIO.UpdateTrackedItem(trackedItem, strconv.Itoa(int(tweet.ID)))
	}

	return nil
}
