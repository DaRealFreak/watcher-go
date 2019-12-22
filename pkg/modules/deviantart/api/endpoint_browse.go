package api

import (
	"net/url"
	"strconv"
)

// BrowseTags contains all relevant information of the API response of the browse function of the tags endpoint
type BrowseTags struct {
	HasMore        bool `json:"has_more"`
	NextOffset     uint `json:"next_offset"`
	EstimatedTotal uint `json:"estimated_total"`
	Results        []struct {
		Author struct {
			UserID   string `json:"userid"`
			Username string `json:"username"`
		} `json:"author"`
		Content struct {
			Src string `json:"src"`
		} `json:"content"`
		DeviationID    string `json:"deviationid"`
		DeviationURL   string `json:"url"`
		Title          string `json:"title"`
		PublishedTime  string `json:"published_time"`
		Excerpt        string `json:"excerpt"`
		IsDownloadable bool   `json:"is_downloadable"`
	} `json:"results"`
}

// BrowseTags implements the API endpoint https://www.deviantart.com/api/v1/oauth2/browse/tags
func (a *DeviantartAPI) BrowseTags(tag string, offset uint, limit uint) (*BrowseTags, error) {
	values := url.Values{
		"tag":    {tag},
		"offset": {strconv.FormatUint(uint64(offset), 10)},
		"limit":  {strconv.FormatUint(uint64(limit), 10)},
	}

	res, err := a.request("GET", "/browse/tags", values)
	if err != nil {
		return nil, err
	}

	var browseTags BrowseTags
	err = a.mapAPIResponse(res, &browseTags)

	return &browseTags, err
}
