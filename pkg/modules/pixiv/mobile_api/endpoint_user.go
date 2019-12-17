package mobileapi

import (
	"encoding/json"
	"net/url"
	"strconv"

	pixivapi "github.com/DaRealFreak/watcher-go/pkg/modules/pixiv/pixiv_api"
)

// UserDetail contains all relevant information regarding the user details
type UserDetail struct {
	User struct {
		ID         json.Number `json:"id"`
		Name       string      `json:"name"`
		IsFollowed bool        `json:"is_followed"`
	} `json:"user"`
	Profile struct {
		Website            string      `json:"webpage"`
		TotalIllustrations json.Number `json:"total_illusts"`
		TotalManga         json.Number `json:"total_manga"`
		TotalNovels        json.Number `json:"total_novels"`
	} `json:"profile"`
}

// GetUserDetail returns the user details from the API
func (a *MobileAPI) GetUserDetail(userID int) (*UserDetail, error) {
	a.ApplyRateLimit()

	apiURL, _ := url.Parse("https://app-api.pixiv.net/v1/user/detail")
	data := url.Values{
		"user_id": {strconv.Itoa(userID)},
	}
	apiURL.RawQuery = data.Encode()

	res, err := a.Session.Get(apiURL.String())
	if err != nil {
		return nil, err
	}

	// user got deleted or deactivated his account
	if res != nil && (res.StatusCode == 403 || res.StatusCode == 404) {
		return nil, pixivapi.UserUnavailableError{
			APIError: pixivapi.APIError{ErrorMessage: "user got either deleted or is unavailable"},
		}
	}

	var userDetail UserDetail
	if err := a.MapAPIResponse(res, &userDetail); err != nil {
		return nil, err
	}

	return &userDetail, nil
}
