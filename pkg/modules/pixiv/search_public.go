package pixiv

import (
	"encoding/json"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	"io/ioutil"
	"net/url"
	"strconv"
)

// parseSearch parses search words using the previous API (no search limitations)
func (m *pixiv) parseSearch(item *models.TrackedItem) (err error) {
	searchWord, err := m.getSearchWordFromURL(item.URI)
	if err != nil {
		return err
	}

	var downloadQueue []*downloadQueueItem
	foundCurrentItem := false
	page := 1

	for !foundCurrentItem {
		response, err := m.getPublicSearch(searchWord, m.getPublicSearchTargetFromURL(item.URI), page)
		if err != nil {
			return err
		}
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

		if response.Pagination.Next != "" {
			if page64, err := response.Pagination.Next.Int64(); err == nil {
				page = int(page64)
			}
		} else {
			// break if we don't have another page
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

// getSearch returns search results directly by URL since the API response returns the next page URL directly
func (m *pixiv) getPublicSearch(word string, searchMode string, page int) (apiRes *publicSearchResponse, err error) {
	var userWorks publicSearchResponse

	apiURL, _ := url.Parse("https://public-api.secure.pixiv.net/v1/search/works.json")
	data := url.Values{
		"q":                    {word},
		"page":                 {strconv.Itoa(page)},
		"per_page":             {"1000"},
		"period":               {"all"},
		"order":                {"desc"},
		"sort":                 {"date"},
		"mode":                 {searchMode},
		"types":                {"illustration,manga,ugoira"},
		"include_stats":        {"true"},
		"include_sanity_level": {"true"},
		"image_sizes":          {"px_128x128,px_480mw,large"},
	}
	apiURL.RawQuery = data.Encode()
	res, err := m.pixivSession.PublicAPI.Get(apiURL.String())
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

// getSearchTargetFromURL returns the search mode based from the passed URI
// default search mode is the partial tag match
func (m *pixiv) getPublicSearchTargetFromURL(uri string) string {
	u, _ := url.Parse(uri)
	q, _ := url.ParseQuery(u.RawQuery)
	// no search mode or partial tag search mod got defined
	if len(q["s_mode"]) == 0 || q["s_mode"][0] == "s_tag" {
		return PublicAPISearchModePartialTagMatch
	}
	// full tag search mode got defined
	if q["s_mode"][0] == "s_tag_full" {
		return PublicAPISearchModeExactTagMatch
	}
	// current web doesn't differentiate between tag and description anymore, so we use text as default
	return PublicAPISearchModeText
}
