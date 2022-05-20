package graphql_api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
)

type UserByScreenNameRequestVariables struct {
	ScreenName                 string `json:"screen_name"`
	WithSafetyModeUserFields   bool   `json:"withSafetyModeUserFields"`
	WithSuperFollowsUserFields bool   `json:"withSuperFollowsUserFields"`
}

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

	jsonString, err := json.Marshal(UserByScreenNameRequestVariables{
		ScreenName:                 username,
		WithSafetyModeUserFields:   true,
		WithSuperFollowsUserFields: true,
	})
	if err != nil {
		return err
	}

	apiURI := "https://twitter.com/i/api/graphql/Bhlf1dYJ3bYCKmLfeEQ31A/UserByScreenName"
	values := url.Values{
		"variables": {string(jsonString)},
	}

	res, err := a.apiGET(apiURI, values)
	if err != nil {
		return err
	}

	tmp, _ := ioutil.ReadAll(res.Body)
	print(tmp)

	/*
		var userInformation *UserInformation
		err = a.mapAPIResponse(res, &userInformation)
	*/

	return err
}
