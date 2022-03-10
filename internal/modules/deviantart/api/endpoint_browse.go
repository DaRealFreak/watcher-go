package api

import (
	"net/url"
	"strconv"
)

// PaginatedResults contains the commonly used pagination of the DeviantArt API
type PaginatedResults struct {
	HasMore        bool         `json:"has_more"`
	NextOffset     *uint        `json:"next_offset"`
	EstimatedTotal uint         `json:"estimated_total"`
	Results        []*Deviation `json:"results"`
}

// Deviation contains all relevant information on artworks/deviations returned from the API
type Deviation struct {
	Author struct {
		UserID   string `json:"userid"`
		Username string `json:"username"`
	} `json:"author"`
	Content *struct {
		Src string `json:"src"`
	} `json:"content"`
	Flash *struct {
		Src string `json:"src"`
	} `json:"flash"`
	Thumbs []struct {
		Src string `json:"src"`
	} `json:"thumbs"`
	Videos []struct {
		Src      string `json:"src"`
		Quality  string `json:"quality"`
		FileSize int    `json:"filesize"`
	} `json:"videos"`
	// used for comparison of thumbs and preview download, not actually returned by the API
	DeviationDownload *DeviationDownload
	DeviationID       string  `json:"deviationid"`
	DeviationURL      string  `json:"url"`
	Title             string  `json:"title"`
	PublishedTime     string  `json:"published_time"`
	Excerpt           *string `json:"excerpt"`
	IsDownloadable    bool    `json:"is_downloadable"`
}

// BrowseTags contains all relevant information of the API response of the browse function of the tags endpoint
type BrowseTags struct {
	PaginatedResults
}

// BrowseNewest contains all relevant information of the API response of the browse function of the newest endpoint
type BrowseNewest struct {
	PaginatedResults
}

// BrowseTags implements the API endpoint https://www.deviantart.com/api/v1/oauth2/browse/tags
func (a *DeviantartAPI) BrowseTags(tag string, offset uint, limit uint) (*BrowseTags, error) {
	values := url.Values{
		"tag":            {tag},
		"offset":         {strconv.FormatUint(uint64(offset), 10)},
		"limit":          {strconv.FormatUint(uint64(limit), 10)},
		"mature_content": {"true"},
	}

	res, err := a.request("GET", "/browse/tags", values)
	if err != nil {
		return nil, err
	}

	var browseTags BrowseTags
	err = a.mapAPIResponse(res, &browseTags)

	return &browseTags, err
}

// BrowseNewest implements the API endpoint https://www.deviantart.com/api/v1/oauth2/browse/newest
func (a *DeviantartAPI) BrowseNewest(query string, offset uint, limit uint) (*BrowseNewest, error) {
	values := url.Values{
		"q":              {query},
		"offset":         {strconv.FormatUint(uint64(offset), 10)},
		"limit":          {strconv.FormatUint(uint64(limit), 10)},
		"mature_content": {"true"},
	}

	res, err := a.request("GET", "/browse/newest", values)
	if err != nil {
		return nil, err
	}

	var browseNewest BrowseNewest
	err = a.mapAPIResponse(res, &browseNewest)

	return &browseNewest, err
}
