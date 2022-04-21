package twitter

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules/twitter/api"
)

func (m *twitter) parsePage(item *models.TrackedItem) error {
	screenName, screenNameErr := m.extractScreenName(item.URI)
	if screenNameErr != nil {
		return screenNameErr
	}

	userInformation, userErr := m.twitterAPI.UserByUsername(screenName)
	if userErr != nil {
		return userErr
	}

	var (
		paginationToken string
		latestTweetID   string
		newMediaTweets  []api.TweetV2
	)

	for {
		tweets, timeLineErr := m.twitterAPI.UserTimelineV2(userInformation.Data.ID.String(), item.CurrentItem, "", paginationToken)
		if timeLineErr != nil {
			return timeLineErr
		}

		if latestTweetID == "" && len(tweets.Data) > 0 {
			latestTweetID = tweets.Data[0].ID.String()
		}

		// parse all media tweets we can find
		for _, tweet := range tweets.Data {
			if strings.HasPrefix(tweet.Text, "RT @") {
				continue
			}

			if len(tweet.Attachments.MediaKeys) > 0 {
				for _, mediaKey := range tweet.Attachments.MediaKeys {
					for _, media := range tweets.Includes.Media {
						if media.MediaKey == mediaKey {
							tweet.Attachments.Media = append(tweet.Attachments.Media, media)
							break
						}
					}
				}

				// attach username to the tweet
				tweet.AuthorName = userInformation.Data.Username

				newMediaTweets = append(newMediaTweets, tweet)
			}
		}

		if len(tweets.Data) == 0 || tweets.Meta.NextToken == "" {
			break
		}

		// set token to check the next page
		paginationToken = tweets.Meta.NextToken
	}

	for i, j := 0, len(newMediaTweets)-1; i < j; i, j = i+1, j-1 {
		newMediaTweets[i], newMediaTweets[j] = newMediaTweets[j], newMediaTweets[i]
	}

	if downloadErr := m.processDownloadQueue(newMediaTweets, item); downloadErr != nil {
		return downloadErr
	}

	if latestTweetID != "" {
		// update also if no new media item got found to possibly save some queries if media tweets happen later
		m.DbIO.UpdateTrackedItem(item, latestTweetID)
	}

	return nil
}

func (m *twitter) extractScreenName(uri string) (string, error) {
	results := regexp.MustCompile(`.*twitter.com/(.*)?(?:$|/)`).FindStringSubmatch(uri)
	if len(results) != 2 {
		return "", fmt.Errorf("unexpected amount of results during screen name extraction of uri %s", uri)
	}

	return results[1], nil
}
