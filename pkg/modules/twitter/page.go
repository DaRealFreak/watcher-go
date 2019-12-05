package twitter

import (
	"fmt"
	"log"
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

	res, apiErr, err := m.getUserTimeline(values)
	if err != nil {
		log.Fatal(err)
		return nil
	}

	if apiErr != nil {
		return fmt.Errorf("api error occurred")
	}

	fmt.Println(res)

	return nil
}

func (m *twitter) extractScreenName(uri string) (string, error) {
	results := regexp.MustCompile(`.*twitter.com/(.*)?(?:$|/)`).FindStringSubmatch(uri)
	if len(results) != 2 {
		return "", fmt.Errorf("unexpected amount of results during screen name extraction of uri %s", uri)
	}

	return results[1], nil
}
