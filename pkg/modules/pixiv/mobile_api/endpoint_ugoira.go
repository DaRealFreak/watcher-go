package mobileapi

import (
	"encoding/json"
	"net/url"
	"strconv"
)

// UgoiraMetadata contains all relevant information regarding the animation details (ugoira)
type UgoiraMetadata struct {
	Metadata struct {
		ZipURLs struct {
			Medium string `json:"medium"`
		} `json:"zip_urls"`
		Frames []struct {
			File  string      `json:"file"`
			Delay json.Number `json:"delay"`
		} `json:"frames"`
	} `json:"ugoira_metadata"`
}

// GetUgoiraMetadata returns the animation details from the API
func (a *MobileAPI) GetUgoiraMetadata(illustID int) (*UgoiraMetadata, error) {
	var ugoiraMetadata UgoiraMetadata

	apiURL, _ := url.Parse("https://app-api.pixiv.net/v1/ugoira/metadata")
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
		return nil, IllustrationUnavailableError{
			APIError{ErrorMessage: "illustration got either deleted or is unavailable"},
		}
	}

	if err := a.mapAPIResponse(res, &ugoiraMetadata); err != nil {
		return nil, err
	}

	return &ugoiraMetadata, nil
}
