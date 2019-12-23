package api

import (
	"net/url"
)

// DeviationContent contains all relevant information of the deviation content response of the API
type DeviationContent struct {
	HTML string `json:"html"`
}

// DeviationContent implements the API endpoint https://www.deviantart.com/api/v1/oauth2/deviation/content
func (a *DeviantartAPI) DeviationContent(deviationID string) (*DeviationContent, error) {
	values := url.Values{
		"deviationid": {deviationID},
	}

	res, err := a.request("GET", "/deviation/content", values)
	if err != nil {
		return nil, err
	}

	var deviationContent DeviationContent
	err = a.mapAPIResponse(res, &deviationContent)

	return &deviationContent, err
}
