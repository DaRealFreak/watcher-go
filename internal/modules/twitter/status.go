package twitter

import (
	"fmt"
	"regexp"

	"github.com/DaRealFreak/watcher-go/internal/modules/twitter/graphql_api"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules/twitter/api"
)

func (m *twitter) parseStatus(item *models.TrackedItem) (err error) {
	screenName, screenNameErr := m.extractStatusID(item.URI)
	if screenNameErr != nil {
		return screenNameErr
	}

	if m.settings.Api.UseGraphQlApi {
		err = m.parseStatusGraphQLApi(item, screenName)
	} else {
		err = m.parseStatusDeveloperApi(item, screenName)
	}

	if err == nil {
		m.DbIO.ChangeTrackedItemCompleteStatus(item, true)
	}

	return err
}

func (m *twitter) parseStatusGraphQLApi(item *models.TrackedItem, statusID string) error {
	tweet, err := m.twitterGraphQlAPI.StatusTweet(statusID)
	if err != nil {
		return err
	}

	var newMediaTweets []*graphql_api.Tweet
	for _, singleTweet := range tweet.TweetEntries() {
		newMediaTweets = append(newMediaTweets, singleTweet)
	}

	return m.processDownloadQueueGraphQL(newMediaTweets, item)
}

func (m *twitter) parseStatusDeveloperApi(item *models.TrackedItem, statusID string) error {
	tweet, err := m.twitterAPI.SingleTweetV2(statusID)
	if err != nil {
		return err
	}

	return m.processDownloadQueueDeveloperApi([]api.TweetV2{*tweet.Data}, item)
}

func (m *twitter) extractStatusID(uri string) (string, error) {
	results := regexp.MustCompile(`.*twitter.com/[^/]+/status/(\d+)`).FindStringSubmatch(uri)

	if len(results) != 2 {
		return "", fmt.Errorf("unexpected amount of results during status ID extraction of uri %s", uri)
	}

	return results[1], nil
}
