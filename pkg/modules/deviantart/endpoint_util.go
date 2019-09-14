package deviantart

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/DaRealFreak/watcher-go/pkg/raven"
)

// Placebo implements the API endpoint https://www.deviantart.com/api/v1/oauth2/placebo
func (m *deviantArt) Placebo() (apiRes *UtilPlaceboResponse, apiErr *APIError) {
	values := url.Values{}
	res, err := m.deviantArtSession.APIPost("/placebo", values, ScopeBasic)
	raven.CheckError(err)

	m.mapAPIResponse(res, &apiRes, &apiErr)
	return apiRes, apiErr
}

// mapAPIResponse maps the API response into the passed APIResponse type
// or into the passed APIError if the status code is 400
func (m *deviantArt) mapAPIResponse(res *http.Response, apiRes interface{}, apiErr interface{}) {
	var reader io.ReadCloser
	switch res.Header.Get("Content-Encoding") {
	case "gzip":
		reader, _ = gzip.NewReader(res.Body)
	default:
		reader = res.Body
	}

	content, err := ioutil.ReadAll(reader)
	raven.CheckError(err)

	if res.StatusCode >= 400 {
		// unmarshal the request content into the error struct
		raven.CheckError(json.Unmarshal(content, &apiErr))
	} else {
		// unmarshal the request content into the response struct
		raven.CheckError(json.Unmarshal(content, &apiRes))
	}
}
