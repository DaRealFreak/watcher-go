package api

import (
	"fmt"
	"net/url"
)

// SingleTweetV1 single tweet lookup
func (a *TwitterAPI) SingleTweetV1(
	tweetId string,
) (*TweetV1, error) {
	a.applyRateLimit()

	apiURI := fmt.Sprintf("https://api.x.com/1.1/statuses/show.json?id=%s", tweetId)
	values := url.Values{
		"tweet_mode": {"extended"},
	}

	res, err := a.apiGET(apiURI, values)
	if err != nil {
		return nil, err
	}

	var tweet *TweetV1
	err = a.mapAPIResponse(res, &tweet)

	return tweet, err
}
