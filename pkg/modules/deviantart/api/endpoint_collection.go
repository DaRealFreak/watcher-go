package api

import (
	"fmt"
	"net/url"
	"strconv"
)

// Collection contains all relevant information of the API response of the collections endpoint
type Collection struct {
	PaginatedResults
}

// Folder contains all available information of the API response regarding to the folder information
type Folder struct {
	FolderUUID string `json:"folderid"`
	Name       string `json:"name"`
}

// Folders contains all relevant information of the API response of the folders function
// of the collection and gallery endpoint
type Folders struct {
	Results    []Folder `json:"results"`
	HasMore    bool     `json:"has_more"`
	NextOffset *int     `json:"next_offset"`
}

// Collection implements the API endpoint https://www.deviantart.com/api/v1/oauth2/collections/{folderid}
func (a *DeviantartAPI) Collection(user string, folderID string, offset uint, limit uint) (*Collection, error) {
	values := url.Values{
		"username":       {user},
		"folderid":       {folderID},
		"offset":         {strconv.Itoa(int(offset))},
		"limit":          {strconv.Itoa(int(limit))},
		"mature_content": {"true"},
	}

	res, err := a.request("GET", "/collections/"+folderID, values)
	if err != nil {
		return nil, err
	}

	var collection Collection
	err = a.mapAPIResponse(res, &collection)

	return &collection, err
}

// CollectionFolders implements the API endpoint https://www.deviantart.com/api/v1/oauth2/collections/folders
func (a *DeviantartAPI) CollectionFolders(user string, offset uint, limit uint) (*Folders, error) {
	values := url.Values{
		"username":       {user},
		"offset":         {strconv.Itoa(int(offset))},
		"limit":          {strconv.Itoa(int(limit))},
		"mature_content": {"true"},
	}

	res, err := a.request("GET", "/collections/folders", values)
	if err != nil {
		return nil, err
	}

	var folders Folders
	err = a.mapAPIResponse(res, &folders)

	return &folders, err
}

// CollectionNameFromURL returns the collection name from the passed URL with the Eclipse theme
func (a *DeviantartAPI) CollectionNameFromURL(feURL string) (string, error) {
	feRes, err := a.Session.Get(feURL)
	if err != nil {
		return "", err
	}

	document := a.Session.GetDocument(feRes)

	return document.Find("div#sub-folder-gallery h2").First().Text(), nil
}

// CollectionNameFromID returns the title of the collection extracted from the frontend, only works with integer IDs
func (a *DeviantartAPI) CollectionNameFromID(username string, folderID int) (string, error) {
	return a.CollectionNameFromURL(
		fmt.Sprintf("https://www.deviantart.com/%s/favourites/%d", username, folderID),
	)
}

// CollectionFolderIDToUUID converts an integer folder ID in combination with the username to the API format folder UUID
// nolint: dupl
func (a *DeviantartAPI) CollectionFolderIDToUUID(username string, folderID int) (string, error) {
	folderName, err := a.CollectionNameFromID(username, folderID)
	if err != nil {
		return "", err
	}

	folderResults, err := a.CollectionFolders(username, 0, MaxDeviationsPerPage)
	if err != nil {
		return "", err
	}

	for _, folder := range folderResults.Results {
		if folder.Name == folderName {
			return folder.FolderUUID, nil
		}
	}

	for folderResults.NextOffset != nil && folderResults.HasMore {
		folderResults, err = a.CollectionFolders(username, uint(*folderResults.NextOffset), MaxDeviationsPerPage)
		if err != nil {
			return "", err
		}

		for _, folder := range folderResults.Results {
			if folder.Name == folderName {
				return folder.FolderUUID, nil
			}
		}
	}

	return "", nil
}
