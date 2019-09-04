package deviantart

import (
	"github.com/DaRealFreak/watcher-go/pkg/raven"
	"net/url"
	"strconv"
)

// Collections implements the API endpoint https://www.deviantart.com/api/v1/oauth2/collections/{folderid}
func (m *deviantArt) Collections(user string, folderID string, offset uint, limit uint) (
	apiRes *CollectionsResponse, apiErr *APIError,
) {
	values := url.Values{
		"username": {user},
		"folderid": {folderID},
		"offset":   {strconv.FormatUint(uint64(offset), 10)},
		"limit":    {strconv.FormatUint(uint64(limit), 10)},
	}

	res, err := m.deviantArtSession.APIGet("/collections/"+folderID, values, ScopeBrowse)
	raven.CheckError(err)

	// map the http.Response into either the api response or the api error
	m.mapAPIResponse(res, &apiRes, &apiErr)
	return apiRes, apiErr
}
