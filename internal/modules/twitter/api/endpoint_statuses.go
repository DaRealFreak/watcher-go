package api

import (
	"fmt"
	"net/url"
	"strconv"
)

// Tweet contains the struct to unmarshal twitters API responses
type Tweet struct {
	CreatedAt        string `json:"created_at"`
	ID               uint   `json:"id"`
	Text             string `json:"text"`
	ExtendedEntities struct {
		Media []*struct {
			ID            uint   `json:"id"`
			MediaURLHTTPS string `json:"media_url_https"`
			DisplayURL    string `json:"display_url"`
			Type          string `json:"type"`
			VideoInfo     struct {
				Variants []*struct {
					Bitrate     uint   `json:"bitrate"`
					ContentType string `json:"content_type"`
					URL         string `json:"url"`
				} `json:"variants"`
			} `json:"video_info"`
		} `json:"media"`
	} `json:"extended_entities"`
	RetweetedStatus interface{} `json:"retweeted_status"`
}

// UserTimeline retrieves the tweets of the passed user, sinceID and maxID will be omitted if an empty string is passed
// API documentation can be found here:
// https://developer.twitter.com/en/docs/tweets/timelines/api-reference/get-statuses-user_timeline
func (a *TwitterAPI) UserTimeline(
	screenName string, sinceID string, maxID string, count uint, includeRetweets bool,
) ([]*Tweet, error) {
	a.applyRateLimit()

	apiURI := "https://api.twitter.com/1.1/statuses/user_timeline.json"
	values := url.Values{
		"screen_name": {screenName},
		"trim_user":   {"1"},
		"count":       {strconv.Itoa(int(count))},
		"include_rts": {fmt.Sprintf("%t", includeRetweets)},
	}

	if sinceID != "" {
		values.Set("since_id", sinceID)
	}

	if maxID != "" {
		values.Set("max_id", maxID)
	}

	res, err := a.apiGET(apiURI, values)
	if err != nil {
		return nil, err
	}

	var userTimeline []*Tweet
	err = a.mapAPIResponse(res, &userTimeline)

	return userTimeline, err
}
