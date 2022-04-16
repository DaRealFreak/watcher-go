package twitter

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules/twitter/api"
)

func (m *twitter) parsePage(item *models.TrackedItem) error {
	screenName, err := m.extractScreenName(item.URI)
	if err != nil {
		return err
	}

	userInformation, err := m.twitterAPI.UserByUsername(screenName)
	if err != nil {
		return err
	}

	var (
		paginationToken string
		latestTweetID   string
		newMediaTweets  []api.TweetV2
	)

	for {
		tweets, err := m.twitterAPI.UserTimelineV2(userInformation.Data.ID.String(), "", "", paginationToken)
		if err != nil {
			return nil
		}

		if latestTweetID == "" && len(tweets.Data) > 0 {
			latestTweetID = tweets.Data[0].ID.String()
		}

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

	if err = m.processDownloadQueue(newMediaTweets, item); err != nil {
		return err
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

// filterMediaTweets returns a filtered amount of tweets having media elements attached since the search endpoint
// only filters the indexed tweets of 6-9 days and is unreliable
func (m *twitter) filterMediaTweets(tweets []*api.TweetV1) (mediaTweets []*api.TweetV1) {
	for _, tweet := range tweets {
		if len(tweet.ExtendedEntities.Media) > 0 {
			mediaTweets = append(mediaTweets, tweet)
		}
	}

	return mediaTweets
}

// filterRetweet is an option to filter retweets from the passed tweets or also original tweets
func (m *twitter) filterRetweet(tweets []*api.TweetV1, retweet bool) (responseTweets []*api.TweetV1) {
	for _, tweet := range tweets {
		if retweet == (tweet.RetweetedStatus.User.ID > 0) {
			responseTweets = append(responseTweets, tweet)
		}
	}

	return responseTweets
}
