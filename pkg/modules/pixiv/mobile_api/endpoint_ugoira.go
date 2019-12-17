package mobileapi

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	pixivapi "github.com/DaRealFreak/watcher-go/pkg/modules/pixiv/pixiv_api"
)

// UgoiraFrame contains all available information of the animation frames
type UgoiraFrame struct {
	File  string      `json:"file"`
	Delay json.Number `json:"delay"`
}

// UgoiraMetadata contains all relevant information regarding the animation details (ugoira)
type UgoiraMetadata struct {
	Metadata struct {
		ZipURLs struct {
			Medium string `json:"medium"`
		} `json:"zip_urls"`
		Frames []*UgoiraFrame `json:"frames"`
	} `json:"ugoira_metadata"`
}

// GetUgoiraFrame returns the associated UgoiraFrame information if existent
func (m *UgoiraMetadata) GetUgoiraFrame(fileName string) (*UgoiraFrame, error) {
	for _, frame := range m.Metadata.Frames {
		if frame.File == fileName {
			return frame, nil
		}
	}

	return nil, fmt.Errorf("no frame found for file: %s", fileName)
}

// GetUgoiraMetadata returns the animation details from the API
func (a *MobileAPI) GetUgoiraMetadata(illustID int) (*UgoiraMetadata, error) {
	a.ApplyRateLimit()

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
		return nil, pixivapi.IllustrationUnavailableError{
			APIError: pixivapi.APIError{ErrorMessage: "illustration got either deleted or is unavailable"},
		}
	}

	var ugoiraMetadata UgoiraMetadata
	if err := a.MapAPIResponse(res, &ugoiraMetadata); err != nil {
		return nil, err
	}

	return &ugoiraMetadata, nil
}
