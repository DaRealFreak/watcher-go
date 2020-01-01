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
func (a *DeviantartAPI) Gallery(user string, folderID string, offset uint, limit uint) (*GalleryResponse, error) {
	values := url.Values{
		"username":       {user},
		"folderid":       {folderID},
		"mode":           {"newest"},
		"offset":         {strconv.Itoa(int(offset))},
		"limit":          {strconv.Itoa(int(limit))},
		"mature_content": {"true"},
	}

	res, err := a.request("GET", "/gallery/"+url.PathEscape(folderID), values)
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
		"username":       {user},
		"offset":         {strconv.Itoa(int(offset))},
		"limit":          {strconv.Itoa(int(limit))},
		"mature_content": {"true"},
	}

	res, err := a.request("GET", "/gallery/all", values)
	if err != nil {
		return nil, err
	}

	var galleryAll GalleryResponse
	err = a.mapAPIResponse(res, &galleryAll)

	return &galleryAll, err
}

// GalleryFolders implements the API endpoint https://www.deviantart.com/api/v1/oauth2/gallery/folders
func (a *DeviantartAPI) GalleryFolders(user string, offset uint, limit uint) (*Folders, error) {
	values := url.Values{
		"username":       {user},
		"offset":         {strconv.Itoa(int(offset))},
		"limit":          {strconv.Itoa(int(limit))},
		"mature_content": {"true"},
	}

	res, err := a.request("GET", "/gallery/folders", values)
	if err != nil {
		return nil, err
	}

	var folders Folders
	err = a.mapAPIResponse(res, &folders)

	return &folders, err
}

// GalleryNameFromID returns the title of the gallery extracted from the frontend, only works with integer IDs
func (a *DeviantartAPI) GalleryNameFromID(username string, folderID int) (string, error) {
	feURL := fmt.Sprintf("https://www.deviantart.com/%s/gallery/%d", username, folderID)

	feRes, err := a.Session.Get(feURL)
	if err != nil {
		return "", err
	}

	return a.Session.GetDocument(feRes).Find("div#sub-folder-gallery h2").First().Text(), nil
}

// GalleryFolderIDToUUID converts an integer folder ID in combination with the username to the API format folder UUID
// nolint: dupl
func (a *DeviantartAPI) GalleryFolderIDToUUID(username string, folderID int) (string, error) {
	folderTitle, err := a.GalleryNameFromID(username, folderID)
	if err != nil {
		return "", err
	}

	folderResults, err := a.GalleryFolders(username, 0, MaxDeviationsPerPage)
	if err != nil {
		return "", err
	}

	for _, folder := range folderResults.Results {
		if folder.Name == folderTitle {
			return folder.FolderUUID, nil
		}
	}

	for folderResults.NextOffset != nil && folderResults.HasMore {
		folderResults, err = a.GalleryFolders(username, uint(*folderResults.NextOffset), MaxDeviationsPerPage)
		if err != nil {
			return "", err
		}

		for _, folder := range folderResults.Results {
			if folder.Name == folderTitle {
				return folder.FolderUUID, nil
			}
		}
	}

	return "", nil
}
