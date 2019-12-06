package twitter

import (
	"fmt"
	"path"

	"github.com/DaRealFreak/watcher-go/pkg/models"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// processDownloadQueue downloads all media entities from the passed tweets if set
func (m *twitter) processDownloadQueue(downloadQueue []*Tweet, trackedItem *models.TrackedItem) error {
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

		for _, entity := range tweet.Entities.MediaElement {
			if err := m.Session.DownloadFile(
				path.Join(
					viper.GetString("download.directory"),
					m.Key,
					screenName,
					fmt.Sprintf("%s_%s", tweet.ID.String(), m.GetFileName(entity.MediaURLHTTPS)),
				),
				entity.MediaURLHTTPS,
			); err != nil {
				return err
			}
		}

		m.DbIO.UpdateTrackedItem(trackedItem, tweet.ID.String())
	}

	return nil
}

func (m *twitter) reverseTweets(tweets []*Tweet) []*Tweet {
	for i, j := 0, len(tweets)-1; i < j; i, j = i+1, j-1 {
		tweets[i], tweets[j] = tweets[j], tweets[i]
	}

	return tweets
}
