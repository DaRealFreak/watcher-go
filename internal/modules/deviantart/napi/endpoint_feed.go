package napi

import (
	"net/url"
)

func (a *DeviantartNAPI) DeviationsFeed(cursor string) (*SearchResponse, error) {
	values := url.Values{
		"csrf_token": {a.CSRFToken},
	}

	if cursor != "" {
		values.Set("cursor", cursor)
	}

	apiUrl := "https://www.deviantart.com/_puppy/dabrowse/networkbar/rfy/deviations?" + values.Encode()
	response, err := a.get(apiUrl)
	if err != nil {
		return nil, err
	}

	var searchResponse SearchResponse
	err = a.mapAPIResponse(response, &searchResponse)

	return &searchResponse, err
}
