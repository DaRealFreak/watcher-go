package deviantart

import (
	"github.com/DaRealFreak/watcher-go/pkg/raven"
	"net/url"
	"strconv"
)

// Deviation implements the API endpoint https://www.deviantart.com/api/v1/oauth2/deviation/{deviationid}
func (m *deviantArt) Deviation(deviationID string) (apiRes *Deviation, apiErr *APIError) {
	res, err := m.deviantArtSession.APIGet(
		"https://www.deviantart.com/api/v1/oauth2/deviation/"+url.QueryEscape(deviationID),
		ScopeBrowse,
	)
	raven.CheckError(err)

	// map the http.Response into either the api response or the api error
	m.mapAPIResponse(res, &apiRes, &apiErr)
	return apiRes, apiErr
}

// DeviationContent implements the API endpoint https://www.deviantart.com/api/v1/oauth2/deviation/content
func (m *deviantArt) DeviationContent(deviationID string) (apiRes *DeviationContent, apiErr *APIError) {
	apiURL, err := url.Parse("https://www.deviantart.com/api/v1/oauth2/deviation/content")
	raven.CheckError(err)

	// add our API values and replace the RawQuery of the apiUrl
	values := url.Values{
		"deviationid": {deviationID},
	}
	apiURL.RawQuery = values.Encode()

	res, err := m.deviantArtSession.APIGet(apiURL.String(), ScopeBrowse)

	// map the http.Response into either the api response or the api error
	m.mapAPIResponse(res, &apiRes, &apiErr)
	return apiRes, apiErr
}

// DeviationDownload implements the API endpoint
// https://www.deviantart.com/api/v1/oauth2/deviation/download/{deviationid}
func (m *deviantArt) DeviationDownload(deviationID string) (apiRes *Image, apiErr *APIError) {
	res, err := m.deviantArtSession.APIGet(
		"https://www.deviantart.com/api/v1/oauth2/deviation/download/"+url.QueryEscape(deviationID),
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
	deviationId string, offsetDeviationID string, offset uint, limit uint,
) (apiRes *EmbeddedContentPagination, apiErr *APIError) {
	apiURL, err := url.Parse("https://www.deviantart.com/api/v1/oauth2/deviation/embeddedcontent")
	raven.CheckError(err)

	// add our API values and replace the RawQuery of the apiUrl
	values := url.Values{
		"deviationid":        {deviationId},
		"offset_deviationid": {offsetDeviationID},
		"offset":             {strconv.FormatUint(uint64(offset), 10)},
		"limit":              {strconv.FormatUint(uint64(limit), 10)},
	}
	apiURL.RawQuery = values.Encode()

	res, err := m.deviantArtSession.APIGet(apiURL.String(), ScopeBrowse)

	// map the http.Response into either the api response or the api error
	m.mapAPIResponse(res, &apiRes, &apiErr)
	return apiRes, apiErr
}
