package napi

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type UserResponse struct {
	HasMore    bool         `json:"hasMore"`
	NextOffset *json.Number `json:"nextOffset"`
	Deviations []*struct {
		Deviation *Deviation `json:"deviation"`
	} `json:"results"`
}

type UserInfo struct {
	User            *Author     `json:"user"`
	DeviationsCount json.Number `json:"deviationsCount"`
}

// UserInfoExpandDefault is the string value for the default expand parameter for the user info
const UserInfoExpandDefault = "user.stats,user.profile,user.watch"

func (a *DeviantartNAPI) UserInfo(username string, expand string) (*UserInfo, error) {
	values := url.Values{
		"username": {username},
	}

	if expand != "" {
		values.Set("expand", expand)
	}

	apiUrl := "https://www.deviantart.com/_napi/shared_api/user/info?" + values.Encode()
	response, err := a.UserSession.Get(apiUrl)
	if err != nil {
		return nil, err
	}

	var userInfo UserInfo
	err = a.mapAPIResponse(response, &userInfo)

	return &userInfo, err
}

func (a *DeviantartNAPI) GalleriesOverviewUser(username string, deviationsLimit int, withSubFolders bool) (*Overview, error) {
	values := url.Values{
		"username":         {username},
		"deviations_limit": {strconv.Itoa(deviationsLimit)},
	}

	if withSubFolders {
		values.Set("with_subfolders", "true")
	}

	apiUrl := "https://www.deviantart.com/_napi/da-user-profile/api/init/gallery?" + values.Encode()
	response, err := a.UserSession.Get(apiUrl)
	if err != nil {
		return nil, err
	}

	var favoritesOverview Overview
	err = a.mapAPIResponse(response, &favoritesOverview)

	return &favoritesOverview, err
}

func (a *DeviantartNAPI) FavoritesOverviewUser(username string, deviationsLimit int, withSubFolders bool) (*Overview, error) {
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

	var favoritesOverview Overview
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
		values.Set("folderid", strconv.Itoa(folder))
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

type WatchResponse struct {
	Success bool `json:"success"`
}

func (a *DeviantartNAPI) WatchUser(username string) (*WatchResponse, error) {
	res, err := a.UserSession.Get(fmt.Sprintf("https://www.deviantart.com/%s", username))
	if err != nil {
		return nil, err
	}

	csrf, csrfErr := a.GetLoginCSRFToken(res)
	if csrfErr != nil {
		return nil, csrfErr
	}

	values := map[string]string{
		"username":   username,
		"csrf_token": csrf.CSRFToken,
	}

	jsonString, _ := json.Marshal(values)
	bodyReader := strings.NewReader(string(jsonString))

	res, err = a.UserSession.GetClient().Post(
		"https://www.deviantart.com/_napi/shared_api/watch",
		"application/json",
		bodyReader,
	)
	if err != nil {
		return nil, err
	}

	var watchResponse WatchResponse
	err = a.mapAPIResponse(res, &watchResponse)

	return &watchResponse, err
}

func (a *DeviantartNAPI) UnwatchUser(username string) (*WatchResponse, error) {
	res, err := a.UserSession.Get(fmt.Sprintf("https://www.deviantart.com/%s", username))
	if err != nil {
		return nil, err
	}

	csrf, csrfErr := a.GetLoginCSRFToken(res)
	if csrfErr != nil {
		return nil, csrfErr
	}

	values := map[string]string{
		"username":   username,
		"csrf_token": csrf.CSRFToken,
	}

	jsonString, _ := json.Marshal(values)
	bodyReader := strings.NewReader(string(jsonString))

	res, err = a.UserSession.GetClient().Post(
		"https://www.deviantart.com/_napi/shared_api/unwatch",
		"application/json",
		bodyReader,
	)
	if err != nil {
		return nil, err
	}

	var watchResponse WatchResponse
	err = a.mapAPIResponse(res, &watchResponse)

	return &watchResponse, err
}
