package publicapi

import (
	"encoding/json"
	"net/url"
	"strconv"
)

// PublicIllustration is the illustration response from the public API search
type PublicIllustration struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Type      string `json:"type"`
	ImageURLs struct {
		Large string `json:"large"`
	} `json:"image_urls"`
	PageCount int `json:"page_count"`
}

// SearchIllust contains all relevant information regarding the illustration search
type SearchIllust struct {
	Illustrations []PublicIllustration `json:"response"`
	Status        string               `json:"status"`
	Count         json.Number          `json:"count"`
	Pagination    struct {
		Previous *int `json:"previous"`
		Current  *int `json:"current"`
		Next     *int `json:"next"`
		Total    *int `json:"total"`
	} `json:"pagination"`
}

//noinspection GoUnusedConst
const (
	// order of the search results
	SearchOrderAscending  = "asc"
	SearchOrderDescending = "desc"

	// search modes
	SearchModePartialTagMatch = "tag"
	SearchModeExactTagMatch   = "exact_tag"
	SearchModeText            = "text"
	SearchModeTitleAndCaption = "caption"
)

// GetSearchIllust returns the illustration search results from the API
func (a *PublicAPI) GetSearchIllust(
	word string, searchMode string, searchOrder string, page int,
) (*SearchIllust, error) {
	apiURL, _ := url.Parse("https://public-api.secure.pixiv.net/v1/search/works.json")
	data := url.Values{
		"q":                    {word},
		"page":                 {strconv.Itoa(page)},
		"per_page":             {"1000"},
		"period":               {"all"},
		"order":                {searchOrder},
		"sort":                 {"date"},
		"mode":                 {searchMode},
		"types":                {"illustration,manga,ugoira"},
		"include_stats":        {"true"},
		"include_sanity_level": {"true"},
		"image_sizes":          {"px_128x128,px_480mw,large"},
	}
	apiURL.RawQuery = data.Encode()

	res, err := a.Session.Get(apiURL.String())
	if err != nil {
		panic(err)
	}

	var searchIllust SearchIllust
	if err := a.MapAPIResponse(res, &searchIllust); err != nil {
		return nil, err
	}

	return &searchIllust, nil
}
