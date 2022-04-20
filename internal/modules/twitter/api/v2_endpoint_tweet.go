package api

import (
	"fmt"
	"net/url"
)

// SingleTweetV2 single tweet lookup
func (a *TwitterAPI) SingleTweetV2(
	tweetId string,
) (*TweetV2, error) {
	a.applyRateLimit()

	apiURI := fmt.Sprintf("https://api.twitter.com/2/tweets/%s", tweetId)
	values := url.Values{}

	res, err := a.apiGET(apiURI, values)
	if err != nil {
		return nil, err
	}

	var tweet *TweetV2
	err = a.mapAPIResponse(res, &tweet)

	return tweet, err
}
