package deviantart

import (
	"net/url"
	"strconv"
)

// Gallery implements the API endpoint https://www.deviantart.com/api/v1/oauth2/gallery/{folderid}
func (m *deviantArt) Gallery(user string, folderID string, mode string, offset uint, limit uint) (
	apiRes *GalleryResponse, apiErr *APIError, err error,
) {
	values := url.Values{
		"username": {user},
		"folderid": {folderID},
		"mode":     {mode},
		"offset":   {strconv.FormatUint(uint64(offset), 10)},
		"limit":    {strconv.FormatUint(uint64(limit), 10)},
	}

	res, err := m.deviantArtSession.APIGet("/gallery/"+folderID, values, ScopeBrowse)
	if err != nil {
		return nil, nil, err
	}

	// map the http.Response into either the api response or the api error
	err = m.mapAPIResponse(res, &apiRes, &apiErr)
	return apiRes, apiErr, err
}

// GalleryAll implements the API endpoint https://www.deviantart.com/api/v1/oauth2/gallery/all
func (m *deviantArt) GalleryAll(user string, offset uint, limit uint) (
	apiRes *GalleryAllResponse, apiErr *APIError, err error,
) {
	values := url.Values{
		"username": {user},
		"offset":   {strconv.FormatUint(uint64(offset), 10)},
		"limit":    {strconv.FormatUint(uint64(limit), 10)},
	}

	res, err := m.deviantArtSession.APIGet("/gallery/all", values, ScopeBrowse)
	if err != nil {
		return nil, nil, err
	}

	// map the http.Response into either the api response or the api error
	err = m.mapAPIResponse(res, &apiRes, &apiErr)
	return apiRes, apiErr, err
}
