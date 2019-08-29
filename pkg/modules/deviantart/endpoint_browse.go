package deviantart

import (
	"encoding/json"
	"io/ioutil"
	"net/url"
	"strconv"

	"github.com/DaRealFreak/watcher-go/pkg/raven"
)

// ScopeBrowse is the required scope for the OAuth2 Token
const ScopeBrowse = "browse"

// BrowseGalleryAll implements the API endpoint https://www.deviantart.com/api/v1/oauth2/gallery/all
func (m *deviantArt) BrowseGalleryAll(username string, offset uint, limit uint) (response *BrowseGalleryAllResponse) {
	var galleryAllResponse BrowseGalleryAllResponse
	apiURL, err := url.Parse("https://www.deviantart.com/api/v1/oauth2/gallery/all")
	raven.CheckError(err)

	// add our API values and replace the RawQuery of the apiUrl
	values := url.Values{
		"username": {username},
		"offset":   {strconv.FormatUint(uint64(offset), 10)},
		"limit":    {strconv.FormatUint(uint64(limit), 10)},
	}
	apiURL.RawQuery = values.Encode()

	res, err := m.deviantArtSession.APIGet(apiURL.String(), ScopeBrowse)
	raven.CheckError(err)

	content, err := ioutil.ReadAll(res.Body)
	raven.CheckError(err)

	// unmarshal the request content into the response struct
	raven.CheckError(json.Unmarshal(content, &galleryAllResponse))
	return &galleryAllResponse
}
