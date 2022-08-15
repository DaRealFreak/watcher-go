package napi

import (
	"encoding/json"
	"net/url"
)

type SearchResponse struct {
	HasMore        bool         `json:"hasMore"`
	NextCursor     string       `json:"nextCursor"`
	EstimatedTotal *json.Number `json:"estTotal"`
	CurrentOffset  *json.Number `json:"currentOffset"`
	Deviations     []*Deviation `json:"deviations"`
}

func (a *DeviantartNAPI) DeviationSearch(search string, cursor string, order string) (*SearchResponse, error) {
	values := url.Values{
		"q": {search},
		// set order to most-recent by default, update if set later
		"order": {"most-recent"},
	}

	if cursor != "" {
		values.Set("cursor", cursor)
	}

	if order != "" {
		values.Set("order", order)
	}

	apiUrl := "https://www.deviantart.com/_napi/da-browse/api/networkbar/search/deviations?" + values.Encode()
	response, err := a.UserSession.Get(apiUrl)
	if err != nil {
		return nil, err
	}

	var searchResponse SearchResponse
	err = a.mapAPIResponse(response, &searchResponse)

	return &searchResponse, err
}

func (a *DeviantartNAPI) DeviationTag(tag string, cursor string, order string) (*SearchResponse, error) {
	values := url.Values{
		"tag": {tag},
		// set order to most-recent by default, update if set later
		"order": {"most-recent"},
	}

	if cursor != "" {
		values.Set("cursor", cursor)
	}

	if order != "" {
		values.Set("order", order)
	}

	apiUrl := "https://www.deviantart.com/_napi/da-browse/api/networkbar/tag/deviations?" + values.Encode()
	response, err := a.UserSession.Get(apiUrl)
	if err != nil {
		return nil, err
	}

	var searchResponse SearchResponse
	err = a.mapAPIResponse(response, &searchResponse)

	return &searchResponse, err
}
