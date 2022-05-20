package graphql_api

import (
	"fmt"
	"net/url"
)

func (a *TwitterGraphQlAPI) UserTimeline(
	userId string, sinceID string, untilId string, paginationToken string,
) error {
	a.applyRateLimit()

	apiURI := fmt.Sprintf("https://api.twitter.com/2/users/%s/tweets", userId)
	values := url.Values{
		"max_results":  {"100"},
		"expansions":   {"attachments.media_keys,author_id"},
		"tweet.fields": {"attachments,author_id,conversation_id,created_at,entities,id,referenced_tweets,text"},
		"media.fields": {"duration_ms,height,media_key,preview_image_url,type,url,width"},
		"user.fields":  {},
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

	_, err := a.apiGET(apiURI, values)
	if err != nil {
		return err
	}

	return err
}

func (a *TwitterGraphQlAPI) UserByUsername(username string) error {
	a.applyRateLimit()

	apiURI := fmt.Sprintf("https://api.twitter.com/2/users/by/username/%s", username)
	values := url.Values{}

	_, err := a.apiGET(apiURI, values)
	if err != nil {
		return err
	}

	/*
		var userInformation *UserInformation
		err = a.mapAPIResponse(res, &userInformation)
	*/

	return err
}
