package napi

import (
	"encoding/json"
	"net/url"
	"strconv"
)

type CollectionsResponse struct {
	HasMore        bool          `json:"hasMore"`
	NextCursor     string        `json:"nextCursor"`
	EstimatedTotal json.Number   `json:"estTotal"`
	CurrentOffset  json.Number   `json:"currentOffset"`
	Collections    []*Collection `json:"collections"`
}

type CollectionsUserResponse struct {
	HasMore     bool          `json:"hasMore"`
	NextOffset  *json.Number  `json:"nextOffset"`
	Collections []*Collection `json:"results"`
}

const FolderTypeGallery = "gallery"
const FolderTypeFavourites = "collection"
const CollectionLimit = 1000

func (a *DeviantartNAPI) CollectionsUser(
	username string, offset int, limit int, folderType string, withSubFolders bool,
) (*CollectionsUserResponse, error) {
	values := url.Values{
		"username": {username},
		"offset":   {strconv.Itoa(offset)},
		"limit":    {strconv.Itoa(limit)},
	}

	if folderType != "" {
		values.Set("type", folderType)
	}

	if withSubFolders {
		values.Set("with_subfolders", "true")
	} else {
		values.Set("with_subfolders", "false")
	}

	apiUrl := "https://www.deviantart.com/_napi/shared_api/gallection/folders?" + values.Encode()
	response, err := a.UserSession.Get(apiUrl)
	if err != nil {
		return nil, err
	}

	var searchResponse CollectionsUserResponse
	err = a.mapAPIResponse(response, &searchResponse)

	return &searchResponse, err
}

func (a *DeviantartNAPI) CollectionSearch(search string, cursor string, order string) (*CollectionsResponse, error) {
	values := url.Values{
		"q": {search},
		// set order to most-recent by default, update if set later
		"order": {OrderMostRecent},
	}

	if cursor != "" {
		values.Set("cursor", cursor)
	}

	if order != "" {
		values.Set("order", order)
	}

	apiUrl := "https://www.deviantart.com/_napi/da-browse/api/networkbar/search/collections?" + values.Encode()
	response, err := a.UserSession.Get(apiUrl)
	if err != nil {
		return nil, err
	}

	var searchResponse CollectionsResponse
	err = a.mapAPIResponse(response, &searchResponse)

	return &searchResponse, err
}
