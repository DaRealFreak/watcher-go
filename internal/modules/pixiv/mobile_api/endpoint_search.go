package mobileapi

import (
	"net/url"
	"strconv"
	"time"
)

// SearchIllust contains all relevant information regarding the illustration search
type SearchIllust struct {
	Illustrations []Illustration `json:"illusts"`
	NextURL       string         `json:"next_url"`
}

//noinspection GoUnusedConst
const (
	// order of the search results
	SearchOrderDateAscending        = "date_asc"
	SearchOrderDateDescending       = "date_desc"
	SearchOrderPopularityAscending  = "popular_asc"
	SearchOrderPopularityDescending = "popular_desc"

	// search modes
	SearchModePartialTagMatch = "partial_match_for_tags"
	SearchModeExactTagMatch   = "exact_match_for_tags"
	SearchModeTitleAndCaption = "title_and_caption"
)

// GetSearchIllust returns the illustration search results from the API
func (a *MobileAPI) GetSearchIllust(
	word string, searchMode string, searchOrder string, offset int,
	minBookmarks int, startDate *time.Time, endDate *time.Time,
) (*SearchIllust, error) {
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

	if minBookmarks > 0 {
		data.Add("bookmark_num_min", strconv.Itoa(minBookmarks))
	}

	if startDate != nil && endDate != nil {
		data.Add("start_date", startDate.Format("2006-01-02"))
		data.Add("end_date", endDate.Format("2006-01-02"))
	}

	apiURL.RawQuery = data.Encode()

	return a.GetSearchIllustByURL(apiURL.String())
}

// GetSearchIllustByURL returns the illustration search results from the API by passed URL
func (a *MobileAPI) GetSearchIllustByURL(url string) (*SearchIllust, error) {
	a.ApplyRateLimit()

	res, err := a.Session.Get(url)
	if err != nil {
		panic(err)
	}

	var searchIllust SearchIllust
	if err := a.MapAPIResponse(res, &searchIllust); err != nil {
		return nil, err
	}

	return &searchIllust, nil
}
