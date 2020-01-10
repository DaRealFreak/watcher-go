package api

import (
	"net/url"
	"strconv"
)

// BucketDeviationSubmitted is the bucket ID for your deviation feed
const BucketDeviationSubmitted = "deviation_submitted"

// FeedBucketResponse contains the next cursor and the items of the current bucket at cursor position
type FeedBucketResponse struct {
	Items []struct {
		Timestamp string `json:"ts"`
		ByUser    struct {
			UserID   string `json:"userid"`
			Username string `json:"username"`
		} `json:"by_user"`
		Deviations []*Deviation `json:"deviations"`
	} `json:"items"`
	Cursor  string `json:"cursor"`
	HasMore bool   `json:"has_more"`
}

// FeedHomeBucket implements the API endpoint https://www.deviantart.com/api/v1/oauth2/feed/home/{bucketid}
func (a *DeviantartAPI) FeedHomeBucket(bucketID string, offset uint) (*FeedBucketResponse, error) {
	// originally there is a "limit" value too, but it is getting completely ignored by the API anyways
	values := url.Values{
		"offset":         {strconv.Itoa(int(offset))},
		"mature_content": {"true"},
	}

	res, err := a.request("GET", "/feed/home/"+url.PathEscape(bucketID), values)
	if err != nil {
		return nil, err
	}

	var feedBucketResponse FeedBucketResponse
	err = a.mapAPIResponse(res, &feedBucketResponse)

	return &feedBucketResponse, err
}

// FeedHome implements the API endpoint https://www.deviantart.com/api/v1/oauth2/feed/home/
func (a *DeviantartAPI) FeedHome(cursor string) (*FeedBucketResponse, error) {
	values := url.Values{
		"cursor":         {cursor},
		"mature_content": {"true"},
	}

	res, err := a.request("GET", "/feed/home/", values)
	if err != nil {
		return nil, err
	}

	var feedBucketResponse FeedBucketResponse
	err = a.mapAPIResponse(res, &feedBucketResponse)

	return &feedBucketResponse, err
}
