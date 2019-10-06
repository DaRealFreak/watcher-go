package deviantart

import (
	"net/url"
	"strconv"
)

// BrowseTags implements the API endpoint https://www.deviantart.com/api/v1/oauth2/browse/tags
func (m *deviantArt) BrowseTags(
	tag string, offset uint, limit uint,
) (apiRes *BrowseTagsResponse, apiErr *APIError, err error) {
	values := url.Values{
		"tag":    {tag},
		"offset": {strconv.FormatUint(uint64(offset), 10)},
		"limit":  {strconv.FormatUint(uint64(limit), 10)},
	}

	res, err := m.deviantArtSession.APIGet("/browse/tags", values, ScopeBrowse)
	if err != nil {
		return nil, nil, err
	}

	// map the http.Response into either the api response or the api error
	err = m.mapAPIResponse(res, &apiRes, &apiErr)
	return apiRes, apiErr, err
}
