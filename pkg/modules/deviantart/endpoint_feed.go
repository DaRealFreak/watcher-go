package deviantart

import (
	"fmt"
	"net/url"
	"strconv"
)

// FeedHomeBucket implements the API endpoint https://www.deviantart.com/api/v1/oauth2/feed/home/{bucketid}
func (m *deviantArt) FeedHomeBucket(bucketID string, offset uint, limit uint, matureContent bool) (
	apiRes *FeedBucketResponse, apiErr *APIError, err error,
) {
	values := url.Values{
		"bucketid":       {bucketID},
		"offset":         {strconv.FormatUint(uint64(offset), 10)},
		"limit":          {strconv.FormatUint(uint64(limit), 10)},
		"mature_content": {fmt.Sprintf("%t", matureContent)},
	}

	res, err := m.deviantArtSession.APIGet("/feed/home/"+bucketID, values, ScopeFeed)
	if err != nil {
		return nil, nil, err
	}

	// map the http.Response into either the api response or the api error
	err = m.mapAPIResponse(res, &apiRes, &apiErr)
	return apiRes, apiErr, err
}

// FeedHomeBucket implements the API endpoint https://www.deviantart.com/api/v1/oauth2/feed/home/
func (m *deviantArt) FeedHome(cursor string, matureContent bool) (
	apiRes *FeedBucketResponse, apiErr *APIError, err error,
) {
	values := url.Values{
		"cursor":         {cursor},
		"mature_content": {fmt.Sprintf("%t", matureContent)},
	}

	res, err := m.deviantArtSession.APIGet("/feed/home/", values, ScopeFeed)
	if err != nil {
		return nil, nil, err
	}

	// map the http.Response into either the api response or the api error
	err = m.mapAPIResponse(res, &apiRes, &apiErr)
	return apiRes, apiErr, err
}
