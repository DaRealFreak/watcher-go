package api

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/PuerkitoBio/goquery"
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

// Folders contains all relevant information of the API response of the folders function of the collection endpoint
type Folders struct {
	Results    []Folder `json:"results"`
	HasMore    bool     `json:"has_more"`
	NextOffset *int     `json:"next_offset"`
}

// Collection implements the API endpoint https://www.deviantart.com/api/v1/oauth2/collections/{folderid}
func (a *DeviantartAPI) Collection(user string, folderID string, offset uint, limit uint) (*Collection, error) {
	values := url.Values{
		"username": {user},
		"folderid": {folderID},
		"offset":   {strconv.Itoa(int(offset))},
		"limit":    {strconv.Itoa(int(limit))},
	}

	res, err := a.request("GET", "/collections/"+folderID, values)
	if err != nil {
		return nil, err
	}

	var collection Collection
	err = a.mapAPIResponse(res, &collection)

	return &collection, err
}

// Folders implements the API endpoint https://www.deviantart.com/api/v1/oauth2/collections/folders
func (a *DeviantartAPI) Folders(user string, offset uint, limit uint) (*Folders, error) {
	values := url.Values{
		"username": {user},
		"offset":   {strconv.Itoa(int(offset))},
		"limit":    {strconv.Itoa(int(limit))},
	}

	res, err := a.request("GET", "/collections/folders", values)
	if err != nil {
		return nil, err
	}

	var folders Folders
	err = a.mapAPIResponse(res, &folders)

	return &folders, err
}

// FolderIDToUUID converts an integer folder ID in combination with the username to the API format folder UUID
func (a *DeviantartAPI) FolderIDToUUID(username string, folderID int) (string, error) {
	feURL := fmt.Sprintf("https://www.deviantart.com/%s/favourites/%d", username, folderID)

	feRes, err := a.Session.Get(feURL)
	if err != nil {
		return "", err
	}

	document, err := goquery.NewDocumentFromReader(feRes.Body)
	if err != nil {
		return "", err
	}

	folderTitle := document.Find("div#sub-folder-gallery h2").First().Text()

	folderResults, err := a.Folders(username, 0, MaxDeviationsPerPage)
	if err != nil {
		return "", err
	}

	for _, folder := range folderResults.Results {
		if folder.Name == folderTitle {
			return folder.FolderUUID, nil
		}
	}

	for folderResults.NextOffset != nil && folderResults.HasMore {
		folderResults, err = a.Folders(username, uint(*folderResults.NextOffset), MaxDeviationsPerPage)
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
