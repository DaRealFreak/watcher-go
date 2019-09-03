package deviantart

import (
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
