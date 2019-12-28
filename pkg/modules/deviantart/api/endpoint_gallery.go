package api

import (
	"net/url"
	"strconv"
)

// GalleryResponse contains all relevant information from the functions of the gallery endpoint
type GalleryResponse struct {
	PaginatedResults
}

// Gallery implements the API endpoint https://www.deviantart.com/api/v1/oauth2/gallery/{folderUUID}
func (a *DeviantartAPI) GalleryByFolderID(user string, folderID int, offset uint, limit uint) (*GalleryResponse, error) {
	folderUUID, err := a.FolderIDToUUID(user, folderID)
	if err != nil {
		return nil, err
	}

	values := url.Values{
		"username": {user},
		"folderid": {folderUUID},
		"mode":     {"newest"},
		"offset":   {strconv.Itoa(int(offset))},
		"limit":    {strconv.Itoa(int(limit))},
	}

	res, err := a.request("GET", "/gallery/"+folderUUID, values)
	if err != nil {
		return nil, err
	}

	var galleryAll GalleryResponse
	err = a.mapAPIResponse(res, &galleryAll)

	return &galleryAll, err
}

// Gallery implements the API endpoint https://www.deviantart.com/api/v1/oauth2/gallery/{folderUUID}
func (a *DeviantartAPI) Gallery(user string, folderUUID string, offset uint, limit uint) (*GalleryResponse, error) {
	values := url.Values{
		"username": {user},
		"folderid": {folderUUID},
		"mode":     {"newest"},
		"offset":   {strconv.Itoa(int(offset))},
		"limit":    {strconv.Itoa(int(limit))},
	}

	res, err := a.request("GET", "/gallery/"+folderUUID, values)
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
