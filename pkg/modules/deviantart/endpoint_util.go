package deviantart

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
)

// Placebo implements the API endpoint https://www.deviantart.com/api/v1/oauth2/placebo
func (m *deviantArt) Placebo() (apiRes *UtilPlaceboResponse, apiErr *APIError, err error) {
	values := url.Values{}
	res, err := m.deviantArtSession.APIPost("/placebo", values, ScopeBasic)
	if err != nil {
		return nil, nil, err
	}

	err = m.mapAPIResponse(res, &apiRes, &apiErr)
	return apiRes, apiErr, err
}

// mapAPIResponse maps the API response into the passed APIResponse type
// or into the passed APIError if the status code is 400
func (m *deviantArt) mapAPIResponse(res *http.Response, apiRes interface{}, apiErr interface{}) (err error) {
	var reader io.ReadCloser
	switch res.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(res.Body)
	default:
		reader = res.Body
	}
	if err != nil {
		return err
	}

	content, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}

	if res.StatusCode >= 400 {
		// unmarshal the request content into the error struct
		if err := json.Unmarshal(content, &apiErr); err != nil {
			return err
		}
	} else {
		// unmarshal the request content into the response struct
		if err := json.Unmarshal(content, &apiRes); err != nil {
			return err
		}
	}
	return nil
}
