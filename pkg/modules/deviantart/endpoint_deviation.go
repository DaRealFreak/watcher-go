package deviantart

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/DaRealFreak/watcher-go/pkg/raven"
)

// Deviation implements the API endpoint https://www.deviantart.com/api/v1/oauth2/deviation/{deviationid}
func (m *deviantArt) Deviation(deviationID string) (apiRes *Deviation, apiErr *APIError) {
	res, err := m.deviantArtSession.APIGet(
		"/deviation/"+url.QueryEscape(deviationID),
		url.Values{},
		ScopeBrowse,
	)
	raven.CheckError(err)

	// map the http.Response into either the api response or the api error
	m.mapAPIResponse(res, &apiRes, &apiErr)
	return apiRes, apiErr
}

// DeviationContent implements the API endpoint https://www.deviantart.com/api/v1/oauth2/deviation/content
func (m *deviantArt) DeviationContent(deviationID string) (apiRes *DeviationContent, apiErr *APIError) {
	values := url.Values{
		"deviationid": {deviationID},
	}

	res, err := m.deviantArtSession.APIGet("/deviation/content", values, ScopeBrowse)
	raven.CheckError(err)

	// map the http.Response into either the api response or the api error
	m.mapAPIResponse(res, &apiRes, &apiErr)
	return apiRes, apiErr
}

// DeviationDownload implements the API endpoint
// https://www.deviantart.com/api/v1/oauth2/deviation/download/{deviationid}
func (m *deviantArt) DeviationDownload(deviationID string) (apiRes *Image, apiErr *APIError) {
	res, err := m.deviantArtSession.APIGet(
		"/deviation/download/"+url.QueryEscape(deviationID),
		url.Values{},
		ScopeBrowse,
	)
	raven.CheckError(err)

	// map the http.Response into either the api response or the api error
	m.mapAPIResponse(res, &apiRes, &apiErr)
	return apiRes, apiErr
}

// DeviationDownloadFallback is a fallback solution for the API endpoint
// https://www.deviantart.com/api/v1/oauth2/deviation/download/{deviationid}
// since the endpoint returns a lot of internal server error responses, while the web interface works properly
func (m *deviantArt) DeviationDownloadFallback(deviationURL string) (apiRes *Image, err error) {
	res, err := m.deviantArtSession.Get(deviationURL)
	raven.CheckError(err)

	doc := m.Session.GetDocument(res)
	dl := doc.Find("a[href*=\"https://www.deviantart.com/download/\"]").First()
	dlLink, exists := dl.Attr("href")
	if dl.Length() != 1 || !exists {
		return nil, fmt.Errorf("no download link found in deviation URL: %s", deviationURL)
	}
	dlWidth, _ := dl.Attr("data-download_width")
	dlHeight, _ := dl.Attr("data-download_height")

	// we can't set Transparency and FileSize without downloading/requesting the file, so we don't return that here
	// it is only a fallback solution after all
	return &Image{
		Src:    dlLink,
		Width:  json.Number(dlWidth),
		Height: json.Number(dlHeight),
	}, nil
}

// DeviationEmbeddedContent implements the API endpoint
// https://www.deviantart.com/api/v1/oauth2/deviation/embeddedcontent
func (m *deviantArt) DeviationEmbeddedContent(
	deviationID string, offsetDeviationID string, offset uint, limit uint,
) (apiRes *EmbeddedContentPagination, apiErr *APIError) {
	values := url.Values{
		"deviationid":        {deviationID},
		"offset_deviationid": {offsetDeviationID},
		"offset":             {strconv.FormatUint(uint64(offset), 10)},
		"limit":              {strconv.FormatUint(uint64(limit), 10)},
	}

	res, err := m.deviantArtSession.APIGet("/deviation/embeddedcontent", values, ScopeBrowse)
	raven.CheckError(err)

	// map the http.Response into either the api response or the api error
	m.mapAPIResponse(res, &apiRes, &apiErr)
	return apiRes, apiErr
}
