package deviantart

import (
	"net/url"
	"strconv"

	"github.com/DaRealFreak/watcher-go/pkg/raven"
)

// GalleryAll implements the API endpoint https://www.deviantart.com/api/v1/oauth2/gallery/all
func (m *deviantArt) GalleryAll(user string, offset uint, limit uint) (apiRes *GalleryAllResponse, apiErr *APIError) {
	apiURL, err := url.Parse("https://www.deviantart.com/api/v1/oauth2/gallery/all")
	raven.CheckError(err)

	// add our API values and replace the RawQuery of the apiUrl
	values := url.Values{
		"username": {user},
		"offset":   {strconv.FormatUint(uint64(offset), 10)},
		"limit":    {strconv.FormatUint(uint64(limit), 10)},
	}
	apiURL.RawQuery = values.Encode()

	res, err := m.deviantArtSession.APIGet(apiURL.String(), ScopeBrowse)
	raven.CheckError(err)

	// map the http.Response into either the api response or the api error
	m.mapAPIResponse(res, &apiRes, &apiErr)
	return apiRes, apiErr
}

// GalleryFoldersCreate implements the API endpoint https://www.deviantart.com/api/v1/oauth2/gallery/folders/create
func (m *deviantArt) GalleryFoldersCreate(folder string) (apiRes *GalleryFoldersCreateResponse, apiErr *APIError) {
	// add our API values and replace the RawQuery of the apiURL
	values := url.Values{
		"folder": {folder},
	}

	res, err := m.deviantArtSession.APIPost(
		"https://www.deviantart.com/api/v1/oauth2/gallery/folders/create",
		values,
		ScopeGallery, ScopeBrowse,
	)
	raven.CheckError(err)

	// map the http.Response into either the api response or the api error
	m.mapAPIResponse(res, &apiRes, &apiErr)
	return apiRes, apiErr
}