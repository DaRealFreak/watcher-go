package napi

import (
	"encoding/json"
	"net/url"
	"strconv"
)

type UserResponse struct {
	HasMore    bool         `json:"hasMore"`
	NextOffset *json.Number `json:"nextOffset"`
	Deviations []*struct {
		Deviation *Deviation `json:"deviation"`
	} `json:"results"`
}

const ModuleNameFolders = "folders"

type FavoritesOverview struct {
	SectionData struct {
		Modules []*struct {
			Name       string `json:"name"`
			ModuleData struct {
				DataKey string `json:"dataKey"`
				Folders struct {
					HasMore    bool        `json:"hasMore"`
					NextOffset json.Number `json:"nextOffset"`
					Results    []*Folder   `json:"results"`
				} `json:"folders"`
			} `json:"moduleData"`
		} `json:"modules"`
	} `json:"sectionData"`
}

func (a *DeviantartNAPI) FavoritesOverviewUser(username string, deviationsLimit int, withSubFolders bool) (*FavoritesOverview, error) {
	values := url.Values{
		"username":         {username},
		"deviations_limit": {strconv.Itoa(deviationsLimit)},
	}

	if withSubFolders {
		values.Set("with_subfolders", "true")
	}

	apiUrl := "https://www.deviantart.com/_napi/da-user-profile/api/init/favourites?" + values.Encode()
	response, err := a.UserSession.Get(apiUrl)
	if err != nil {
		return nil, err
	}

	var favoritesOverview FavoritesOverview
	err = a.mapAPIResponse(response, &favoritesOverview)

	return &favoritesOverview, err
}

func (a *DeviantartNAPI) FavoritesUser(username string, folder int, offset int, limit int, allFolders bool) (*UserResponse, error) {
	values := url.Values{
		"username": {username},
		"offset":   {strconv.Itoa(offset)},
		"limit":    {strconv.Itoa(limit)},
	}

	if folder > 0 {
		values.Set("folderid", strconv.Itoa(folder))
	}

	// if no folder is passed "all_folder" should be true as it is from the original API
	// (no "folder" or "all_folder" argument passed returned the same results as "all_folder=true")
	if folder == 0 || allFolders {
		values.Set("all_folder", "true")
	}

	apiUrl := "https://www.deviantart.com/_napi/da-user-profile/api/collection/contents?" + values.Encode()
	response, err := a.UserSession.Get(apiUrl)
	if err != nil {
		return nil, err
	}

	var userResponse UserResponse
	err = a.mapAPIResponse(response, &userResponse)

	return &userResponse, err
}

func (a *DeviantartNAPI) DeviationsUser(username string, folder int, offset int, limit int, allFolders bool) (*UserResponse, error) {
	values := url.Values{
		"username": {username},
		"offset":   {strconv.Itoa(offset)},
		"limit":    {strconv.Itoa(limit)},
	}

	if folder > 0 {
		values.Set("folder", strconv.Itoa(folder))
	}

	// if no folder is passed "all_folder" should be true as it is from the original API
	// (no "folder" or "all_folder" argument passed returned the same results as "all_folder=true")
	if folder == 0 || allFolders {
		values.Set("all_folder", "true")
	}

	apiUrl := "https://www.deviantart.com/_napi/da-user-profile/api/gallery/contents?" + values.Encode()
	response, err := a.UserSession.Get(apiUrl)
	if err != nil {
		return nil, err
	}

	var userResponse UserResponse
	err = a.mapAPIResponse(response, &userResponse)

	return &userResponse, err
}
