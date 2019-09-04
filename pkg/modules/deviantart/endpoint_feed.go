package deviantart

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/DaRealFreak/watcher-go/pkg/raven"
)

// FeedHomeBucket implements the API endpoint https://www.deviantart.com/api/v1/oauth2/feed/home/{bucketid}
func (m *deviantArt) FeedHomeBucket(bucketID string, offset uint, limit uint, matureContent bool) (
	apiRes *FeedBucketResponse, apiErr *APIError,
) {
	values := url.Values{
		"bucketid":       {bucketID},
		"offset":         {strconv.FormatUint(uint64(offset), 10)},
		"limit":          {strconv.FormatUint(uint64(limit), 10)},
		"mature_content": {fmt.Sprintf("%t", matureContent)},
	}

	res, err := m.deviantArtSession.APIGet("/feed/home/"+bucketID, values, ScopeFeed)
	raven.CheckError(err)

	// map the http.Response into either the api response or the api error
	m.mapAPIResponse(res, &apiRes, &apiErr)
	return apiRes, apiErr
}
