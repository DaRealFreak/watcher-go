package api

import (
	"encoding/json"
	"fmt"
	"net/url"
)

type TweetV2 struct {
	ID          json.Number `json:"id"`
	Text        string      `json:"text"`
	AuthorID    json.Number `json:"author_id"`
	Attachments struct {
		MediaKeys []string `json:"media_keys"`
		Media     []*TweetMedia
	} `json:"attachments"`
}

type TweetMedia struct {
	MediaKey string `json:"media_key"`
	URL      string `json:"url"`
	Type     string `json:"type"`
}

// Tweets contains the struct to unmarshal twitters API responses
type Tweets struct {
	Data     []TweetV2 `json:"data"`
	Includes struct {
		Media []*TweetMedia `json:"media"`
	} `json:"includes"`
	Meta struct {
		NextToken   string      `json:"next_token"`
		ResultCount json.Number `json:"result_count"`
		NewestId    json.Number `json:"newest_id"`
		OldestId    json.Number `json:"oldest_id"`
	} `json:"meta"`
}

type UserInformation struct {
	Data struct {
		ID       json.Number `json:"id"`
		Name     string      `json:"name"`
		Username string      `json:"username"`
	} `json:"data"`
}

// UserTimelineV2 retrieves the timeline of the passed user ID
func (a *TwitterAPI) UserTimelineV2(
	userId string, sinceID string, untilId string, paginationToken string,
) (*Tweets, error) {
	a.applyRateLimit()

	apiURI := fmt.Sprintf("https://api.twitter.com/2/users/%s/tweets", userId)
	values := url.Values{
		"max_results":  {"100"},
		"expansions":   {"attachments.media_keys"},
		"tweet.fields": {"attachments,author_id,conversation_id,created_at,entities,id,referenced_tweets,text"},
		"media.fields": {"duration_ms,height,media_key,preview_image_url,type,url,width"},
	}

	if sinceID != "" {
		values.Set("since_id", sinceID)
	}

	if untilId != "" {
		values.Set("until_id", untilId)
	}

	if paginationToken != "" {
		values.Set("pagination_token", paginationToken)
	}

	res, err := a.apiGET(apiURI, values)
	if err != nil {
		return nil, err
	}

	var tweets *Tweets
	err = a.mapAPIResponse(res, &tweets)

	return tweets, err
}

func (a *TwitterAPI) UserByUsername(username string) (*UserInformation, error) {
	a.applyRateLimit()

	apiURI := fmt.Sprintf("https://api.twitter.com/2/users/by/username/%s", username)
	values := url.Values{}

	res, err := a.apiGET(apiURI, values)
	if err != nil {
		return nil, err
	}

	var userInformation *UserInformation
	err = a.mapAPIResponse(res, &userInformation)

	return userInformation, err
}
