package mobileapi

import (
	"net/url"
	"strconv"
	"time"

	pixivapi "github.com/DaRealFreak/watcher-go/internal/modules/pixiv/pixiv_api"
)

// Illustration contains all relevant information of an illustration
type Illustration struct {
	ID             int    `json:"id"`
	Title          string `json:"title"`
	Type           string `json:"type"`
	Caption        string `json:"caption"`
	MetaSinglePage struct {
		OriginalImageURL *string `json:"original_image_url"`
	} `json:"meta_single_page"`
	MetaPages []*struct {
		ImageURLs struct {
			Original string `json:"original"`
		} `json:"image_urls"`
	} `json:"meta_pages"`
	User       UserInfo  `json:"user"`
	CreateDate time.Time `json:"create_date"`
}

// IllustDetail contains all relevant information regarding an illustration detail API request
type IllustDetail struct {
	Illustration Illustration `json:"illust"`
}

// GetIllustDetail returns the illustration details from the API
func (a *MobileAPI) GetIllustDetail(illustID int) (*IllustDetail, error) {
	a.ApplyRateLimit()

	apiURL, _ := url.Parse("https://app-api.pixiv.net/v1/illust/detail")
	data := url.Values{
		"illust_id": {strconv.Itoa(illustID)},
	}
	apiURL.RawQuery = data.Encode()

	res, err := a.Get(apiURL.String())
	if err != nil {
		return nil, err
	}

	// user got deleted or deactivated his account
	if res != nil && (res.StatusCode == 403 || res.StatusCode == 404) {
		return nil, pixivapi.IllustrationUnavailableError{
			APIError: pixivapi.APIError{ErrorMessage: "illustration got either deleted or is unavailable"},
		}
	}

	var illustDetail IllustDetail
	if err = a.MapAPIResponse(res, &illustDetail); err != nil {
		return nil, err
	}

	return &illustDetail, nil
}
