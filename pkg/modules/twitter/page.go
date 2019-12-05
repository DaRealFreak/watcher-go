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

	tweets, apiErr, err := m.getUserTimeline(values)
	if err != nil {
		return nil
	}

	if apiErr != nil {
		return fmt.Errorf("api error occurred")
	}

	// only find original tweets with media elements attached to it
	mediaTweets := m.filterRetweet(m.filterMediaTweets(tweets), false)

	fmt.Println(mediaTweets)

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
