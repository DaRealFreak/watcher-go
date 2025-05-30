package graphql_api

import (
	"encoding/json"
	"net/url"
)

type StatusTweet struct {
	Data struct {
		ThreadedConversationWithInjectionsV2 *Timeline `json:"threaded_conversation_with_injections_v2"`
	} `json:"data"`
}

func (t *StatusTweet) TweetEntries() (tweets []*Tweet) {
	return t.Data.ThreadedConversationWithInjectionsV2.TweetEntries()
}

func (a *TwitterGraphQlAPI) StatusTweet(
	tweetID string,
) (*StatusTweet, error) {
	a.applyRateLimit()

	variables := map[string]interface{}{
		"focalTweetId":                           tweetID,
		"referrer":                               "profile",
		"with_rux_injections":                    false,
		"includePromotedContent":                 true,
		"withCommunity":                          true,
		"withQuickPromoteEligibilityTweetFields": true,
		"withBirdwatchNotes":                     true,
		"withVoice":                              true,
		"withV2Timeline":                         true,
	}

	variablesJson, _ := json.Marshal(variables)

	featuresJson, _ := json.Marshal(map[string]interface{}{
		"responsive_web_graphql_exclude_directive_enabled":                        true,
		"verified_phone_label_enabled":                                            false,
		"responsive_web_home_pinned_timelines_enabled":                            true,
		"creator_subscriptions_tweet_preview_api_enabled":                         true,
		"responsive_web_graphql_timeline_navigation_enabled":                      true,
		"responsive_web_graphql_skip_user_profile_image_extensions_enabled":       false,
		"c9s_tweet_anatomy_moderator_badge_enabled":                               true,
		"tweetypie_unmention_optimization_enabled":                                true,
		"responsive_web_edit_tweet_api_enabled":                                   true,
		"graphql_is_translatable_rweb_tweet_is_translatable_enabled":              true,
		"view_counts_everywhere_api_enabled":                                      true,
		"longform_notetweets_consumption_enabled":                                 true,
		"responsive_web_twitter_article_tweet_consumption_enabled":                false,
		"tweet_awards_web_tipping_enabled":                                        false,
		"freedom_of_speech_not_reach_fetch_enabled":                               true,
		"standardized_nudges_misinfo":                                             true,
		"tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled": true,
		"longform_notetweets_rich_text_read_enabled":                              true,
		"longform_notetweets_inline_media_enabled":                                true,
		"responsive_web_media_download_video_enabled":                             false,
		"responsive_web_enhance_cards_enabled":                                    false,
	})

	apiURI := "https://x.com/i/api/graphql/BbmLpxKh8rX8LNe2LhVujA/TweetDetail"
	values := url.Values{
		"variables": {string(variablesJson)},
		"features":  {string(featuresJson)},
	}

	res, err := a.handleGetRequest(apiURI, values)
	if err != nil {
		return nil, err
	}

	var timeline *StatusTweet
	err = a.mapAPIResponse(res, &timeline)

	return timeline, err
}
