package mobileapi

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/DaRealFreak/watcher-go/pkg/models"
	pixivapi "github.com/DaRealFreak/watcher-go/pkg/modules/pixiv/pixiv_api"
)

// UserInfo contains the ID, displayed name and the follow status
type UserInfo struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	IsFollowed bool   `json:"is_followed"`
}

// UserDetail contains all relevant information regarding the user details
type UserDetail struct {
	User    UserInfo `json:"user"`
	Profile struct {
		Website            string `json:"webpage"`
		TotalIllustrations int    `json:"total_illusts"`
		TotalManga         int    `json:"total_manga"`
		TotalNovels        int    `json:"total_novels"`
	} `json:"profile"`
}

// GetUserTag returns the default download tag for illustrations of the user context
func (u *UserInfo) GetUserTag() string {
	return fmt.Sprintf("%d/%s", u.ID, models.Module{}.SanitizePath(u.Name, false))
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
