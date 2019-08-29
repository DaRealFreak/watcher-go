package deviantart

import (
	"encoding/json"
	"io/ioutil"
	"net/url"

	"github.com/DaRealFreak/watcher-go/pkg/raven"
)

// BrowseGalleryAll implements the API endpoint https://www.deviantart.com/api/v1/oauth2/browse/categorytree
func (m *deviantArt) BrowseCategoryTree(categoryPath string) *BrowseCategoryTreeResponse {
	var browseCategoryTreeResponse BrowseCategoryTreeResponse
	apiURL, err := url.Parse("https://www.deviantart.com/api/v1/oauth2/browse/categorytree")
	raven.CheckError(err)

	// add our API values and replace the RawQuery of the apiUrl
	values := url.Values{
		"catpath": {categoryPath},
	}
	apiURL.RawQuery = values.Encode()

	res, err := m.deviantArtSession.APIGet(apiURL.String(), ScopeBrowse)
	raven.CheckError(err)

	content, err := ioutil.ReadAll(res.Body)
	raven.CheckError(err)

	// unmarshal the request content into the response struct
	raven.CheckError(json.Unmarshal(content, &browseCategoryTreeResponse))
	return &browseCategoryTreeResponse
}
