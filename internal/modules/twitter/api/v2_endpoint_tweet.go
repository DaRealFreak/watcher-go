package api

import (
	"fmt"
	"net/url"
)

type SingleTweetV2 struct {
	Data     *TweetV2 `json:"data"`
	Includes struct {
		Media []*TweetMedia `json:"media"`
	} `json:"includes"`
}

// SingleTweetV2 single tweet lookup
func (a *TwitterAPI) SingleTweetV2(
	tweetId string,
) (*SingleTweetV2, error) {
	a.applyRateLimit()

	apiURI := fmt.Sprintf("https://api.twitter.com/2/tweets/%s", tweetId)
	values := url.Values{
		"expansions":   {"attachments.media_keys,author_id"},
		"tweet.fields": {"attachments,author_id,conversation_id,created_at,entities,id,referenced_tweets,text"},
		"media.fields": {"duration_ms,height,media_key,preview_image_url,type,url,width"},
		"user.fields":  {},
	}

	res, err := a.apiGET(apiURI, values)
	if err != nil {
		return nil, err
	}

	var tweet *SingleTweetV2
	err = a.mapAPIResponse(res, &tweet)

	return tweet, err
}
