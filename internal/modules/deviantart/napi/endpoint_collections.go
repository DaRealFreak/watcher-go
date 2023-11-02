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
	username string, offset int, limit int, folderType string, withSubFolders bool, includeAllFolder bool,
) (*CollectionsUserResponse, error) {
	values := url.Values{
		"username":   {username},
		"offset":     {strconv.Itoa(offset)},
		"limit":      {strconv.Itoa(limit)},
		"type":       {folderType},
		"csrf_token": {a.csrfToken},
	}

	if withSubFolders {
		values.Set("with_subfolders", "true")
	} else {
		values.Set("with_subfolders", "false")
	}

	apiUrl := "https://www.deviantart.com/_puppy/dashared/gallection/folders?" + values.Encode()
	response, err := a.UserSession.Get(apiUrl)
	if err != nil {
		return nil, err
	}

	var searchResponse CollectionsUserResponse
	if err = a.mapAPIResponse(response, &searchResponse); err != nil {
		return &searchResponse, err
	}

	// unofficial option since the "All" folder is not an official folder but returned in the user profile initialisation
	// since it is used in f.e. author update and contains the folder size we merge both API responses
	if includeAllFolder {
		var allCollection *Collection
		switch folderType {
		case FolderTypeGallery:
			favorites, favoritesErr := a.GalleriesOverviewUser(username, MaxLimit, false)
			if favoritesErr != nil {
				return &searchResponse, favoritesErr
			}

			allCollection = favorites.FindFolderByFolderId(FolderIdAllFolder)
		case FolderTypeFavourites:
			favorites, favoritesErr := a.FavoritesOverviewUser(username, MaxLimit, false)
			if favoritesErr != nil {
				return &searchResponse, favoritesErr
			}

			allCollection = favorites.FindFolderByFolderId(FolderIdAllFolder)
		}

		searchResponse.Collections = append([]*Collection{allCollection}, searchResponse.Collections...)
	}

	return &searchResponse, err
}

func (a *DeviantartNAPI) CollectionSearch(search string, cursor string, order string) (*CollectionsResponse, error) {
	values := url.Values{
		"q": {search},
		// set order to most-recent by default, update if set later
		"order":      {OrderMostRecent},
		"csrf_token": {a.csrfToken},
	}

	if cursor != "" {
		values.Set("cursor", cursor)
	}

	if order != "" {
		values.Set("order", order)
	}

	apiUrl := "https://www.deviantart.com/_puppy/dabrowse/search/collections?" + values.Encode()
	response, err := a.UserSession.Get(apiUrl)
	if err != nil {
		return nil, err
	}

	var searchResponse CollectionsResponse
	err = a.mapAPIResponse(response, &searchResponse)

	return &searchResponse, err
}
