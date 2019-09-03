package deviantart

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/DaRealFreak/watcher-go/pkg/raven"
)

// BrowseGalleryAll implements the API endpoint https://www.deviantart.com/api/v1/oauth2/browse/categorytree
func (m *deviantArt) BrowseCategoryTree(categoryPath string) (apiRes *BrowseCategoryTreeResponse, apiErr *APIError) {
	values := url.Values{
		"catpath": {categoryPath},
	}

	res, err := m.deviantArtSession.APIGet("/browse/categorytree", values, ScopeBrowse)
	raven.CheckError(err)

	// map the http.Response into either the api response or the api error
	m.mapAPIResponse(res, &apiRes, &apiErr)
	return apiRes, apiErr
}

// BrowseDailyDeviations implements the API endpoint https://www.deviantart.com/api/v1/oauth2/browse/dailydeviations
func (m *deviantArt) BrowseDailyDeviations(date string) (apiRes *BrowseDailyDeviationsResponse, apiErr *APIError) {
	values := url.Values{
		"date": {date},
	}

	res, err := m.deviantArtSession.APIGet("/browse/dailydeviations", values, ScopeBrowse)
	raven.CheckError(err)

	// map the http.Response into either the api response or the api error
	m.mapAPIResponse(res, &apiRes, &apiErr)
	return apiRes, apiErr
}

// BrowseHot implements the API endpoint https://www.deviantart.com/api/v1/oauth2/browse/hot
func (m *deviantArt) BrowseHot(catPath string, offset uint, limit uint) (apiRes *BrowseHotResponse, apiErr *APIError) {
	values := url.Values{
		"category_path": {catPath},
		"offset":        {strconv.FormatUint(uint64(offset), 10)},
		"limit":         {strconv.FormatUint(uint64(limit), 10)},
	}

	res, err := m.deviantArtSession.APIGet("/browse/hot", values, ScopeBrowse)
	raven.CheckError(err)

	// map the http.Response into either the api response or the api error
	m.mapAPIResponse(res, &apiRes, &apiErr)
	return apiRes, apiErr
}

// BrowseMoreLikeThis implements the API endpoint https://www.deviantart.com/api/v1/oauth2/browse/morelikethis
func (m *deviantArt) BrowseMoreLikeThis(
	seed string, category string, offset uint, limit uint,
) (apiRes *BrowseMoreLikeThisResponse, apiErr *APIError) {
	values := url.Values{
		"seed":     {seed},
		"category": {category},
		"offset":   {strconv.FormatUint(uint64(offset), 10)},
		"limit":    {strconv.FormatUint(uint64(limit), 10)},
	}

	res, err := m.deviantArtSession.APIGet("/browse/morelikethis", values, ScopeBrowse)
	raven.CheckError(err)

	// map the http.Response into either the api response or the api error
	m.mapAPIResponse(res, &apiRes, &apiErr)
	return apiRes, apiErr
}

// BrowseMoreLikeThisPreview implements the API endpoint
// https://www.deviantart.com/api/v1/oauth2/browse/morelikethis/preview
func (m *deviantArt) BrowseMoreLikeThisPreview(
	seed string,
) (apiRes *BrowseMoreLikeThisPreviewResponse, apiErr *APIError) {
	values := url.Values{
		"seed": {seed},
	}

	res, err := m.deviantArtSession.APIGet("/morelikethis/preview", values, ScopeBrowse)
	raven.CheckError(err)

	// map the http.Response into either the api response or the api error
	m.mapAPIResponse(res, &apiRes, &apiErr)
	return apiRes, apiErr
}

// BrowseNewest implements the API endpoint https://www.deviantart.com/api/v1/oauth2/browse/newest
func (m *deviantArt) BrowseNewest(
	categoryPath string, searchQuery string, offset uint, limit uint,
) (apiRes *BrowseNewestResponse, apiErr *APIError) {
	values := url.Values{
		"category_path": {categoryPath},
		"q":             {searchQuery},
		"offset":        {strconv.FormatUint(uint64(offset), 10)},
		"limit":         {strconv.FormatUint(uint64(limit), 10)},
	}

	res, err := m.deviantArtSession.APIGet("/browse/newest", values, ScopeBrowse)
	raven.CheckError(err)

	// map the http.Response into either the api response or the api error
	m.mapAPIResponse(res, &apiRes, &apiErr)
	return apiRes, apiErr
}

// BrowseNewest implements the API endpoint https://www.deviantart.com/api/v1/oauth2/browse/popular
func (m *deviantArt) BrowsePopular(
	categoryPath string, searchQuery string, timeRange string, offset uint, limit uint,
) (apiRes *BrowseNewestResponse, apiErr *APIError) {
	values := url.Values{
		"category_path": {categoryPath},
		"q":             {searchQuery},
		"timerange":     {timeRange},
		"offset":        {strconv.FormatUint(uint64(offset), 10)},
		"limit":         {strconv.FormatUint(uint64(limit), 10)},
	}

	res, err := m.deviantArtSession.APIGet("/browse/popular", values, ScopeBrowse)
	raven.CheckError(err)

	// map the http.Response into either the api response or the api error
	m.mapAPIResponse(res, &apiRes, &apiErr)
	return apiRes, apiErr
}

// BrowseNewest implements the API endpoint https://www.deviantart.com/api/v1/oauth2/browse/tags
func (m *deviantArt) BrowseTags(
	tag string, offset uint, limit uint,
) (apiRes *BrowseTagsResponse, apiErr *APIError) {
	values := url.Values{
		"tag":    {tag},
		"offset": {strconv.FormatUint(uint64(offset), 10)},
		"limit":  {strconv.FormatUint(uint64(limit), 10)},
	}

	res, err := m.deviantArtSession.APIGet("/browse/tags", values, ScopeBrowse)
	raven.CheckError(err)

	// map the http.Response into either the api response or the api error
	m.mapAPIResponse(res, &apiRes, &apiErr)
	return apiRes, apiErr
}

// BrowseNewest implements the API endpoint https://www.deviantart.com/api/v1/oauth2/browse/tags/search
func (m *deviantArt) BrowseTagsSearch(tagName string) (apiRes *BrowseTagsSearchResponse, apiErr *APIError) {
	values := url.Values{
		"tag_name": {tagName},
	}

	res, err := m.deviantArtSession.APIGet("/tags/search", values, ScopeBrowse)
	raven.CheckError(err)

	// map the http.Response into either the api response or the api error
	m.mapAPIResponse(res, &apiRes, &apiErr)
	return apiRes, apiErr
}

// BrowseNewest implements the API endpoint https://www.deviantart.com/api/v1/oauth2/browse/undiscovered
func (m *deviantArt) BrowseUndiscovered(
	categoryPath string, offset uint, limit uint,
) (apiRes *BrowseUndiscoveredResponse, apiErr *APIError) {
	values := url.Values{
		"category_path": {categoryPath},
		"offset":        {strconv.FormatUint(uint64(offset), 10)},
		"limit":         {strconv.FormatUint(uint64(limit), 10)},
	}

	res, err := m.deviantArtSession.APIGet("/browse/undiscovered", values, ScopeBrowse)
	raven.CheckError(err)

	// map the http.Response into either the api response or the api error
	m.mapAPIResponse(res, &apiRes, &apiErr)
	return apiRes, apiErr
}

// BrowseNewest implements the API endpoint https://www.deviantart.com/api/v1/oauth2/browse/user/journals
func (m *deviantArt) BrowserUserJournals(
	username string, featured bool, offset uint, limit uint,
) (apiRes *BrowseUserJournalsResponse, apiErr *APIError) {
	// add our API values and replace the RawQuery of the apiUrl
	values := url.Values{
		"username": {username},
		"featured": {fmt.Sprintf("%t", featured)},
		"offset":   {strconv.FormatUint(uint64(offset), 10)},
		"limit":    {strconv.FormatUint(uint64(limit), 10)},
	}

	res, err := m.deviantArtSession.APIGet("/browse/user/journals", values, ScopeBrowse)
	raven.CheckError(err)

	// map the http.Response into either the api response or the api error
	m.mapAPIResponse(res, &apiRes, &apiErr)
	return apiRes, apiErr
}
