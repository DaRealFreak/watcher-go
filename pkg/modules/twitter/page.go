package twitter

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"regexp"

	"github.com/DaRealFreak/watcher-go/pkg/models"
)

func (m *twitter) parsePage(item *models.TrackedItem) error {
	screenName, err := m.extractScreenName(item.URI)
	if err != nil {
		return err
	}

	resp, err := m.Session.Get(fmt.Sprintf(
		"https://api.twitter.com/1.1/statuses/user_timeline.json?screen_name=%s&trim_user=true&count=200",
		url.QueryEscape(screenName),
	))
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error: %s", err)
	}

	fmt.Println(string(body))

	return nil
}

func (m *twitter) extractScreenName(uri string) (string, error) {
	results := regexp.MustCompile(`.*twitter.com/(.*)?(?:$|/)`).FindStringSubmatch(uri)
	if len(results) != 2 {
		return "", fmt.Errorf("unexpected amount of results during screen name extraction of uri %s", uri)
	}

	return results[1], nil
}
