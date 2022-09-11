package graphql_api

import (
	"encoding/json"
	"net/url"
)

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
		"withBirdwatchNotes":                     false,
		"withSuperFollowsUserFields":             true,
		"withDownvotePerspective":                false,
		"withReactionsMetadata":                  false,
		"withReactionsPerspective":               false,
		"withSuperFollowsTweetFields":            true,
		"withVoice":                              true,
		"withV2Timeline":                         true,
	}

	variablesJson, _ := json.Marshal(variables)

	featuresJson, _ := json.Marshal(map[string]interface{}{
		"responsive_web_graphql_timeline_navigation_enabled":                      false,
		"unified_cards_ad_metadata_container_dynamic_card_content_query_enabled":  false,
		"dont_mention_me_view_api_enabled":                                        true,
		"responsive_web_uc_gql_enabled":                                           true,
		"vibe_api_enabled":                                                        true,
		"responsive_web_edit_tweet_api_enabled":                                   true,
		"graphql_is_translatable_rweb_tweet_is_translatable_enabled":              false,
		"standardized_nudges_misinfo":                                             true,
		"tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled": false,
		"interactive_text_enabled":                                                true,
		"responsive_web_text_conversations_enabled":                               false,
		"responsive_web_enhance_cards_enabled":                                    true,
	})

	apiURI := "https://twitter.com/i/api/graphql/Nze3idtpjn4wcl09GpmDRg/TweetDetail"
	values := url.Values{
		"variables": {string(variablesJson)},
		"features":  {string(featuresJson)},
	}

	res, err := a.apiGET(apiURI, values)
	if err != nil {
		return nil, err
	}

	var timeline *StatusTweet
	err = a.mapAPIResponse(res, &timeline)

	return timeline, err
}

func (t StatusTweet) TweetEntries() (tweets []*Tweet) {
	for _, instruction := range t.Data.ThreadedConversationWithInjectionsV2.Instructions {
		if instruction.Type != "TimelineAddEntries" {
			continue
		}

		for _, entry := range instruction.Entries {
			if entry.Content.EntryType != "TimelineTimelineItem" {
				continue
			}

			tweets = append(tweets, entry)
		}
	}

	return tweets
}
