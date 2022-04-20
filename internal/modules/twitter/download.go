package twitter

import (
	"fmt"
	"path"
	"strconv"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules/twitter/api"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// processDownloadQueue downloads all media entities from the passed tweets if set
func (m *twitter) processDownloadQueue(downloadQueue []api.TweetV2, trackedItem *models.TrackedItem) error {
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

		for _, media := range tweet.Attachments.Media {
			if media.Type == "video" {
				tweetId, err := tweet.ID.Int64()
				if err != nil {
					return err
				}

				tweetV1, err := m.twitterAPI.UserTimeline(
					tweet.AuthorID.String(),
					strconv.Itoa(int(tweetId-1)),
					strconv.Itoa(int(tweetId)),
					1,
					true,
				)

				for _, entity := range tweetV1[0].ExtendedEntities.Media {
					if entity.Type == "video" {
						highestBitRateIndex := 0
						var highestBitRate uint = 0
						for bitRateIndex, variant := range entity.VideoInfo.Variants {
							if variant.Bitrate >= highestBitRate {
								highestBitRateIndex = bitRateIndex
								highestBitRate = variant.Bitrate
							}
						}

						if err = m.twitterAPI.Session.DownloadFile(
							path.Join(
								viper.GetString("download.directory"),
								m.Key,
								screenName,
								fmt.Sprintf("%s_%s_%s", tweet.ID, tweet.AuthorID.String(), m.GetFileName(entity.VideoInfo.Variants[highestBitRateIndex].URL)),
							),
							entity.VideoInfo.Variants[highestBitRateIndex].URL,
						); err != nil {
							return err
						}
					}
				}
			} else {
				if err := m.twitterAPI.Session.DownloadFile(
					path.Join(
						viper.GetString("download.directory"),
						m.Key,
						screenName,
						fmt.Sprintf("%s_%s_%s", tweet.ID, tweet.AuthorID.String(), m.GetFileName(media.URL)),
					),
					media.URL,
				); err != nil {
					return err
				}
			}
		}

		m.DbIO.UpdateTrackedItem(trackedItem, tweet.ID.String())
	}

	return nil
}
