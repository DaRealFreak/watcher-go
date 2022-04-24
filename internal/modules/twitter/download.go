package twitter

import (
	"fmt"
	"path"

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

	for index, tweet := range downloadQueue {
		log.WithField("module", m.Key).Info(
			fmt.Sprintf(
				"downloading updates for uri: \"%s\" (%0.2f%%)",
				trackedItem.URI,
				float64(index+1)/float64(len(downloadQueue))*100,
			),
		)

		for _, media := range tweet.Attachments.Media {
			if media.Type == "video" || media.Type == "animated_gif" {
				tweetV1, singleTweetErr := m.twitterAPI.SingleTweetV1(
					tweet.ID.String(),
				)
				if singleTweetErr != nil {
					return singleTweetErr
				}

				if tweetV1 == nil || tweetV1.ID == 0 {
					log.WithField("module", m.ModuleKey()).Warnf(
						"unable to retrieve %s of https://twitter.com/%s/status/%s from twitter API v1, skipping tweet",
						media.Type,
						tweet.AuthorName,
						tweet.ID.String(),
					)
					continue
				}

				for _, entity := range tweetV1.ExtendedEntities.Media {
					if entity.Type == "video" || entity.Type == "animated_gif" {
						highestBitRateIndex := 0
						highestBitRate := uint(0)
						for bitRateIndex, variant := range entity.VideoInfo.Variants {
							if variant.Bitrate >= highestBitRate {
								highestBitRateIndex = bitRateIndex
								highestBitRate = variant.Bitrate
							}
						}

						if err := m.twitterAPI.Session.DownloadFile(
							path.Join(
								viper.GetString("download.directory"),
								m.Key,
								tweet.AuthorName,
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
						tweet.AuthorName,
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
