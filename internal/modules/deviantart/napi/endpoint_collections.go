package napi

import (
	"encoding/json"
	"net/url"
)

type CollectionsResponse struct {
	HasMore        bool          `json:"hasMore"`
	NextCursor     string        `json:"nextCursor"`
	EstimatedTotal json.Number   `json:"estTotal"`
	CurrentOffset  json.Number   `json:"currentOffset"`
	Collections    []*Collection `json:"collections"`
}

func (a *DeviantartNAPI) CollectionSearch(search string, cursor string, order string) (*CollectionsResponse, error) {
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

	apiUrl := "https://www.deviantart.com/_napi/da-browse/api/networkbar/search/collections?" + values.Encode()
	response, err := a.UserSession.Get(apiUrl)
	if err != nil {
		return nil, err
	}

	var searchResponse CollectionsResponse
	err = a.mapAPIResponse(response, &searchResponse)

	return &searchResponse, err
}
