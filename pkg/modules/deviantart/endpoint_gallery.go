package deviantart

import (
	"net/url"
	"strconv"

	"github.com/DaRealFreak/watcher-go/pkg/raven"
)

// Gallery implements the API endpoint https://www.deviantart.com/api/v1/oauth2/gallery/{folderid}
func (m *deviantArt) Gallery(user string, folderID string, mode string, offset uint, limit uint) (
	apiRes *GalleryResponse, apiErr *APIError,
) {
	values := url.Values{
		"username": {user},
		"folderid": {folderID},
		"mode":     {mode},
		"offset":   {strconv.FormatUint(uint64(offset), 10)},
		"limit":    {strconv.FormatUint(uint64(limit), 10)},
	}

	res, err := m.deviantArtSession.APIGet("/gallery/"+folderID, values, ScopeBrowse)
	raven.CheckError(err)

	// map the http.Response into either the api response or the api error
	m.mapAPIResponse(res, &apiRes, &apiErr)
	return apiRes, apiErr
}

// GalleryAll implements the API endpoint https://www.deviantart.com/api/v1/oauth2/gallery/all
func (m *deviantArt) GalleryAll(user string, offset uint, limit uint) (apiRes *GalleryAllResponse, apiErr *APIError) {
	values := url.Values{
		"username": {user},
		"offset":   {strconv.FormatUint(uint64(offset), 10)},
		"limit":    {strconv.FormatUint(uint64(limit), 10)},
	}

	res, err := m.deviantArtSession.APIGet("/gallery/all", values, ScopeBrowse)
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
		"/gallery/folders/create",
		values,
		ScopeGallery, ScopeBrowse,
	)
	raven.CheckError(err)

	// map the http.Response into either the api response or the api error
	m.mapAPIResponse(res, &apiRes, &apiErr)
	return apiRes, apiErr
}
