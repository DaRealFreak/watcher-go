package pixiv

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"strconv"

	"github.com/DaRealFreak/watcher-go/pkg/models"
)

// parseSearchApp parses search words for the current API (search limit of 5000 results)
func (m *pixiv) parseSearchApp(item *models.TrackedItem) (err error) {
	searchWord, err := m.getSearchWordFromURL(item.URI)
	if err != nil {
		return err
	}

	var downloadQueue []*downloadQueueItem
	foundCurrentItem := false
	apiURL := m.getSearchURL(searchWord, m.getSearchTargetFromURL(item.URI), SearchOrderDateDescending, 0)

	for !foundCurrentItem {
		response, err := m.getSearch(apiURL)
		if err != nil {
			return err
		}
		apiURL = response.NextURL
		for _, userIllustration := range response.Illustrations {
			if string(userIllustration.ID) == item.CurrentItem {
				foundCurrentItem = true
				break
			}
			err = m.parseWork(userIllustration, &downloadQueue)
			if err != nil {
				return err
			}
		}

		// break if we don't have another page
		if apiURL == "" {
			break
		}
	}

	// reverse download queue to download old items first
	for i, j := 0, len(downloadQueue)-1; i < j; i, j = i+1, j-1 {
		downloadQueue[i], downloadQueue[j] = downloadQueue[j], downloadQueue[i]
	}

	// update download tag to our search word
	for _, queueItem := range downloadQueue {
		queueItem.DownloadTag = searchWord
	}
	return m.processDownloadQueue(downloadQueue, item)
}

// getSearchURL builds the search URL manually
func (m *pixiv) getSearchURL(word string, searchMode string, searchOrder string, offset int) string {
	apiURL, _ := url.Parse("https://app-api.pixiv.net/v1/search/illust")
	data := url.Values{
		"include_translated_tag_results": {"true"},
		"merge_plain_keyword_results":    {"true"},
		"word":                           {word},
		"sort":                           {searchOrder},
		"search_target":                  {searchMode},
	}
	if offset > 0 {
		data.Add("offset", strconv.Itoa(offset))
	}
	apiURL.RawQuery = data.Encode()
	return apiURL.String()
}

// getSearch returns search results directly by URL since the API response returns the next page URL directly
func (m *pixiv) getSearch(apiURL string) (apiRes *searchResponse, err error) {
	var userWorks searchResponse
	res, err := m.Session.Get(apiURL)
	if err != nil {
		return nil, err
	}

	response, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(response, &userWorks)
	return &userWorks, err
}

// getSearchWordFromURL extracts the search word from the passed item URI
func (m *pixiv) getSearchWordFromURL(uri string) (string, error) {
	u, _ := url.Parse(uri)
	q, _ := url.ParseQuery(u.RawQuery)
	if len(q["word"]) == 0 {
		return "", fmt.Errorf("parsed uri(%s) does not contain any \"word\" tag", uri)
	}
	return q["word"][0], nil
}

// getSearchTargetFromURL returns the search mode based from the passed URI
// default search mode is the partial tag match
func (m *pixiv) getSearchTargetFromURL(uri string) string {
	u, _ := url.Parse(uri)
	q, _ := url.ParseQuery(u.RawQuery)
	// no search mode or partial tag search mod got defined
	if len(q["s_mode"]) == 0 || q["s_mode"][0] == "s_tag" {
		return SearchModePartialTagMatch
	}
	// full tag search mode got defined
	if q["s_mode"][0] == "s_tag_full" {
		return SearchModeExactTagMatch
	}
	// last possible option
	return SearchModeTitleAndCaption
}
