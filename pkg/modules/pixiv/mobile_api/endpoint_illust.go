package mobileapi

import (
	"encoding/json"
	"net/url"
	"strconv"
)

// IllustDetail contains all relevant information regarding the illustration details
type IllustDetail struct {
	Illustration struct {
		ID             json.Number `json:"id"`
		Title          string      `json:"title"`
		Type           string      `json:"type"`
		MetaSinglePage struct {
			OriginalImageURL string `json:"original_image_url"`
		} `json:"meta_single_page"`
		// ToDo: mapping instead of map
		MetaPages []map[string]map[string]string `json:"meta_pages"`
	} `json:"illust"`
}

// GetIllustDetail returns the illustration details from the API
func (a *MobileAPI) GetIllustDetail(illustID int) (*IllustDetail, error) {
	var illustDetail IllustDetail

	apiURL, _ := url.Parse("https://app-api.pixiv.net/v1/illust/detail")
	data := url.Values{
		"illust_id": {strconv.Itoa(illustID)},
	}
	apiURL.RawQuery = data.Encode()

	res, err := a.Session.Get(apiURL.String())
	if err != nil {
		return nil, err
	}

	// user got deleted or deactivated his account
	if res != nil && (res.StatusCode == 403 || res.StatusCode == 404) {
		return nil, DeletedIllustrationError{APIError{ErrorMessage: "Illustration could not be found"}}
	}

	if err := a.mapAPIResponse(res, &illustDetail); err != nil {
		return nil, err
	}

	return &illustDetail, nil
}
