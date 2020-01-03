package twitter

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/DaRealFreak/watcher-go/pkg/modules/twitter/api"
)

func (m *twitter) parsePage(item *models.TrackedItem) error {
	screenName, err := m.extractScreenName(item.URI)
	if err != nil {
		return err
	}

	var (
		maxID          string
		latestTweetID  *uint
		newMediaTweets []*api.Tweet
	)

	for {
		tweets, err := m.twitterAPI.UserTimeline(
			screenName, item.CurrentItem, maxID, api.MaxTweetsPerRequest, true,
		)
		if err != nil {
			return nil
		}

		if maxID != "" && len(tweets) > 0 {
			// remove the first element which is our current max_id
			tweets = tweets[1:]
		}

		if latestTweetID == nil && len(tweets) > 0 {
			latestTweetID = &tweets[0].ID
		}

		mediaTweets := m.filterRetweet(m.filterMediaTweets(tweets), false)
		newMediaTweets = append(newMediaTweets, mediaTweets...)

		if len(tweets) < 1 {
			break
		}

		maxID = strconv.Itoa(int(tweets[len(tweets)-1].ID))
	}

	for i, j := 0, len(newMediaTweets)-1; i < j; i, j = i+1, j-1 {
		newMediaTweets[i], newMediaTweets[j] = newMediaTweets[j], newMediaTweets[i]
	}

	if err := m.processDownloadQueue(newMediaTweets, item); err != nil {
		return err
	}

	if latestTweetID != nil {
		// update also if no new media item got found to possibly save some queries if media tweets happen later
		m.DbIO.UpdateTrackedItem(item, strconv.Itoa(int(*latestTweetID)))
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
func (m *twitter) filterMediaTweets(tweets []*api.Tweet) (mediaTweets []*api.Tweet) {
	for _, tweet := range tweets {
		if len(tweet.ExtendedEntities.Media) > 0 {
			mediaTweets = append(mediaTweets, tweet)
		}
	}

	return mediaTweets
}

// filterRetweet is an option to filter retweets from the passed tweets or also original tweets
func (m *twitter) filterRetweet(tweets []*api.Tweet, retweet bool) (responseTweets []*api.Tweet) {
	for _, tweet := range tweets {
		if retweet == (tweet.RetweetedStatus != nil) {
			responseTweets = append(responseTweets, tweet)
		}
	}

	return responseTweets
}
