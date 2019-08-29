package deviantart

import (
	"net/url"

	"github.com/DaRealFreak/watcher-go/pkg/raven"
)

// BrowseGalleryAll implements the API endpoint https://www.deviantart.com/api/v1/oauth2/browse/categorytree
func (m *deviantArt) BrowseCategoryTree(categoryPath string) (*BrowseCategoryTreeResponse, *APIError) {
	apiRes := (*BrowseCategoryTreeResponse)(nil)
	apiErr := (*APIError)(nil)

	apiURL, err := url.Parse("https://www.deviantart.com/api/v1/oauth2/browse/categorytree")
	raven.CheckError(err)

	// add our API values and replace the RawQuery of the apiUrl
	values := url.Values{
		"catpath": {categoryPath},
	}
	apiURL.RawQuery = values.Encode()

	res, err := m.deviantArtSession.APIGet(apiURL.String(), ScopeBrowse)
	raven.CheckError(err)

	// map the http.Response into either the api response or the api error
	m.mapAPIResponse(res, &apiRes, &apiErr)
	return apiRes, apiErr
}
