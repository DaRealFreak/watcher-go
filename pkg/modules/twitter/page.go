package twitter

import (
	"fmt"
	"net/url"
	"regexp"

	"github.com/DaRealFreak/watcher-go/pkg/models"
)

func (m *twitter) parsePage(item *models.TrackedItem) error {
	screenName, err := m.extractScreenName(item.URI)
	if err != nil {
		return err
	}

	values := url.Values{
		"screen_name": {screenName},
		"trim_user":   {"1"},
		"count":       {"200"},
		"include_rts": {"1"},
	}
	maxID := ""
	latestTweetID := ""

	var newMetaTweets []*Tweet

	for {
		if maxID != "" {
			values.Set("max_id", maxID)
		}

		if item.CurrentItem != "" {
			values.Set("since_id", item.CurrentItem)
		}

		tweets, apiErr, err := m.getUserTimeline(values)
		if err != nil {
			return nil
		}

		if apiErr != nil {
			return fmt.Errorf("api error occurred")
		}

		if maxID != "" {
			// remove the first element which is our current max_id
			tweets = tweets[1:]
		} else {
			latestTweetID = tweets[0].ID.String()
		}

		mediaTweets := m.filterRetweet(m.filterMediaTweets(tweets), false)
		newMetaTweets = append(newMetaTweets, mediaTweets...)

		// break if we don't have at least one new tweet to navigate to the next page
		if len(tweets) < 1 {
			break
		}

		maxID = tweets[len(tweets)-1].ID.String()
	}

	// download meta from tweets
	fmt.Println(newMetaTweets)

	// set to the latest tweet ID
	fmt.Println(latestTweetID)

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
func (m *twitter) filterMediaTweets(tweets []*Tweet) (mediaTweets []*Tweet) {
	for _, tweet := range tweets {
		if len(tweet.Entities.MediaElement) > 0 {
			mediaTweets = append(mediaTweets, tweet)
		}
	}

	return mediaTweets
}

// filterRetweet is an option to filter retweets from the passed tweets or also original tweets
func (m *twitter) filterRetweet(tweets []*Tweet, retweet bool) (responseTweets []*Tweet) {
	for _, tweet := range tweets {
		if retweet == (tweet.RetweetedStatus != nil) {
			responseTweets = append(responseTweets, tweet)
		}
	}

	return responseTweets
}
