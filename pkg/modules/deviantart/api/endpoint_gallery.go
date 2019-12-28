package api

import (
	"fmt"
	"net/url"
	"strconv"
)

// GalleryResponse contains all relevant information from the functions of the gallery endpoint
type GalleryResponse struct {
	PaginatedResults
}

// Gallery implements the API endpoint https://www.deviantart.com/api/v1/oauth2/gallery/{folderID}
func (a *DeviantartAPI) Gallery(user string, folderID uint, offset uint, limit uint) (*GalleryResponse, error) {
	values := url.Values{
		"username": {user},
		"folderid": {strconv.Itoa(int(folderID))},
		"mode":     {"newest"},
		"offset":   {strconv.Itoa(int(offset))},
		"limit":    {strconv.Itoa(int(limit))},
	}

	res, err := a.request("GET", fmt.Sprintf("/gallery/%d", folderID), values)
	if err != nil {
		return nil, err
	}

	var galleryAll GalleryResponse
	err = a.mapAPIResponse(res, &galleryAll)

	return &galleryAll, err
}

// GalleryAll implements the API endpoint https://www.deviantart.com/api/v1/oauth2/gallery/all
func (a *DeviantartAPI) GalleryAll(user string, offset uint, limit uint) (*GalleryResponse, error) {
	values := url.Values{
		"username": {user},
		"offset":   {strconv.Itoa(int(offset))},
		"limit":    {strconv.Itoa(int(limit))},
	}

	res, err := a.request("GET", "/gallery/all", values)
	if err != nil {
		return nil, err
	}

	var galleryAll GalleryResponse
	err = a.mapAPIResponse(res, &galleryAll)

	return &galleryAll, err
}
