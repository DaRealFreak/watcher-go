package deviantart

import (
	"net/url"
	"strconv"
)

// Collections implements the API endpoint https://www.deviantart.com/api/v1/oauth2/collections/{folderid}
func (m *deviantArt) Collections(user string, folderID string, offset uint, limit uint) (
	apiRes *CollectionsResponse, apiErr *APIError, err error,
) {
	values := url.Values{
		"username": {user},
		"folderid": {folderID},
		"offset":   {strconv.FormatUint(uint64(offset), 10)},
		"limit":    {strconv.FormatUint(uint64(limit), 10)},
	}

	res, err := m.deviantArtSession.APIGet("/collections/"+folderID, values, ScopeBrowse)
	if err != nil {
		return nil, nil, err
	}

	// map the http.Response into either the api response or the api error
	err = m.mapAPIResponse(res, &apiRes, &apiErr)
	return apiRes, apiErr, err
}
