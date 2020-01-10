package api

import (
	"fmt"
	"net/url"
)

// DeviationContent contains all relevant information of the deviation content response of the API
type DeviationContent struct {
	HTML string `json:"html"`
}

// DeviationDownload contains all relevant information of the deviation download response of the API
type DeviationDownload struct {
	Src string `json:"src"`
}

// DeviationContent implements the API endpoint https://www.deviantart.com/api/v1/oauth2/deviation/content
func (a *DeviantartAPI) DeviationContent(deviationID string) (*DeviationContent, error) {
	values := url.Values{
		"deviationid":    {deviationID},
		"mature_content": {"true"},
	}

	res, err := a.request("GET", "/deviation/content", values)
	if err != nil {
		return nil, err
	}

	var deviationContent DeviationContent
	err = a.mapAPIResponse(res, &deviationContent)

	return &deviationContent, err
}

// DeviationDownload implements the API endpoint
// https://www.deviantart.com/api/v1/oauth2/deviation/download/{deviationid}
func (a *DeviantartAPI) DeviationDownload(deviationID string) (*DeviationDownload, error) {
	res, err := a.request("GET", "/deviation/download/"+url.QueryEscape(deviationID), url.Values{})
	if err != nil {
		return nil, err
	}

	var deviationDownload DeviationDownload
	err = a.mapAPIResponse(res, &deviationDownload)

	return &deviationDownload, err
}

// DeviationDownloadFallback is a fallback solution for the API endpoint
// https://www.deviantart.com/api/v1/oauth2/deviation/download/{deviationid}
// since the endpoint returns a lot of internal server error responses, while the web interface works properly
func (a *DeviantartAPI) DeviationDownloadFallback(deviationURL string) (*DeviationDownload, error) {
	res, err := a.Session.Get(deviationURL)
	if err != nil {
		return nil, err
	}

	dl := a.Session.GetDocument(res).Find(`a[href*="https://www.deviantart.com/download/"]`).First()
	dlLink, exists := dl.Attr("href")

	if dl.Length() != 1 || !exists {
		return nil, fmt.Errorf("no download link found in deviation URL: %s", deviationURL)
	}

	return &DeviationDownload{Src: dlLink}, nil
}
