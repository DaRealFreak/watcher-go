package api

import (
	"fmt"
	"net/url"
	"strconv"
)

// TweetV1 contains the struct to unmarshal twitters API responses
type TweetV1 struct {
	CreatedAt        string `json:"created_at"`
	ID               uint   `json:"id"`
	Text             string `json:"full_text"`
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
	RetweetedStatus struct {
		User struct {
			ID uint `json:"id"`
		} `json:"user"`
	} `json:"retweeted_status"`
}

// UserTimeline retrieves the tweets of the passed user, sinceID and maxID will be omitted if an empty string is passed
// API documentation can be found here:
// https://developer.twitter.com/en/docs/tweets/timelines/api-reference/get-statuses-user_timeline
func (a *TwitterAPI) UserTimeline(
	userId string, sinceID string, maxID string, count uint, includeRetweets bool,
) ([]*TweetV1, error) {
	a.applyRateLimit()

	apiURI := "https://api.twitter.com/1.1/statuses/user_timeline.json"
	values := url.Values{
		"tweet_mode":  {"extended"},
		"user_id":     {userId},
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

	var userTimeline []*TweetV1
	err = a.mapAPIResponse(res, &userTimeline)

	return userTimeline, err
}
