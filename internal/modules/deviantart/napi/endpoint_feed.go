package napi

import (
	"net/url"
)

func (a *DeviantartNAPI) DeviationsFeed(cursor string) (*SearchResponse, error) {
	values := url.Values{}

	if cursor != "" {
		values.Set("cursor", cursor)
	}

	apiUrl := "https://www.deviantart.com/_napi/da-browse/api/networkbar/rfy/deviations?" + values.Encode()
	response, err := a.UserSession.Get(apiUrl)
	if err != nil {
		return nil, err
	}

	var searchResponse SearchResponse
	err = a.mapAPIResponse(response, &searchResponse)

	return &searchResponse, err
}
