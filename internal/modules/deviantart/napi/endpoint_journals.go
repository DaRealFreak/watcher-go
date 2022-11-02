package napi

import (
	"net/url"
)

func (a *DeviantartNAPI) JournalSearch(search string, cursor string, order string) (*SearchResponse, error) {
	values := url.Values{
		"q": {search},
		// set order to most-recent by default, update if set later
		"order":      {OrderMostRecent},
		"csrf_token": {a.csrfToken},
	}

	if cursor != "" {
		values.Set("cursor", cursor)
	}

	if order != "" {
		values.Set("order", order)
	}

	apiUrl := "https://www.deviantart.com/_napi/da-browse/api/networkbar/search/journals?" + values.Encode()
	response, err := a.UserSession.Get(apiUrl)
	if err != nil {
		return nil, err
	}

	var searchResponse SearchResponse
	err = a.mapAPIResponse(response, &searchResponse)

	return &searchResponse, err
}

func (a *DeviantartNAPI) JournalTag(tag string, cursor string, order string) (*SearchResponse, error) {
	values := url.Values{
		"tag": {tag},
		// set order to most-recent by default, update if set later
		"order":      {OrderMostRecent},
		"csrf_token": {a.csrfToken},
	}

	if cursor != "" {
		values.Set("cursor", cursor)
	}

	if order != "" {
		values.Set("order", order)
	}

	apiUrl := "https://www.deviantart.com/_napi/da-browse/api/networkbar/tag/journals?" + values.Encode()
	response, err := a.UserSession.Get(apiUrl)
	if err != nil {
		return nil, err
	}

	var searchResponse SearchResponse
	err = a.mapAPIResponse(response, &searchResponse)

	return &searchResponse, err
}
