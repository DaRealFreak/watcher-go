package graphql_api

import (
	"encoding/json"
	"net/url"
)

type StatusTweet struct {
	Data struct {
		Thread struct {
			Instructions []Instruction[TweetEntry] `json:"instructions"`
		} `json:"threaded_conversation_with_injections_v2"`
	} `json:"data"`
}

func (t *StatusTweet) TweetEntries() (tweets []*Tweet) {
	for _, instruction := range t.Data.Thread.Instructions {
		if instruction.Type != "TimelineAddEntries" {
			continue
		}

		for _, entry := range instruction.Entries {
			tweets = append(tweets, entry.Content.GetTweet())
		}

	}
	return tweets
}

func (a *TwitterGraphQlAPI) StatusTweet(
	tweetID string,
) (*StatusTweet, error) {
	a.applyRateLimit()

	variables := map[string]interface{}{
		"focalTweetId":                           tweetID,
		"referrer":                               "profile",
		"with_rux_injections":                    false,
		"rankingMode":                            "Relevance",
		"includePromotedContent":                 true,
		"withCommunity":                          true,
		"withQuickPromoteEligibilityTweetFields": true,
		"withBirdwatchNotes":                     true,
		"withVoice":                              true,
	}
	varsJSON, err := json.Marshal(variables)
	if err != nil {
		return nil, err
	}

	features := map[string]interface{}{
		"rweb_video_screen_enabled":                                               false,
		"payments_enabled":                                                        false,
		"profile_label_improvements_pcf_label_in_post_enabled":                    true,
		"rweb_tipjar_consumption_enabled":                                         true,
		"verified_phone_label_enabled":                                            false,
		"creator_subscriptions_tweet_preview_api_enabled":                         true,
		"responsive_web_graphql_timeline_navigation_enabled":                      true,
		"responsive_web_graphql_skip_user_profile_image_extensions_enabled":       false,
		"premium_content_api_read_enabled":                                        false,
		"communities_web_enable_tweet_community_results_fetch":                    true,
		"c9s_tweet_anatomy_moderator_badge_enabled":                               true,
		"responsive_web_grok_analyze_button_fetch_trends_enabled":                 false,
		"responsive_web_grok_analyze_post_followups_enabled":                      true,
		"responsive_web_jetfuel_frame":                                            false,
		"responsive_web_grok_share_attachment_enabled":                            true,
		"articles_preview_enabled":                                                true,
		"responsive_web_edit_tweet_api_enabled":                                   true,
		"graphql_is_translatable_rweb_tweet_is_translatable_enabled":              true,
		"view_counts_everywhere_api_enabled":                                      true,
		"longform_notetweets_consumption_enabled":                                 true,
		"responsive_web_twitter_article_tweet_consumption_enabled":                true,
		"tweet_awards_web_tipping_enabled":                                        false,
		"responsive_web_grok_show_grok_translated_post":                           false,
		"responsive_web_grok_analysis_button_from_backend":                        true,
		"creator_subscriptions_quote_tweet_preview_enabled":                       false,
		"freedom_of_speech_not_reach_fetch_enabled":                               true,
		"standardized_nudges_misinfo":                                             true,
		"tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled": true,
		"longform_notetweets_rich_text_read_enabled":                              true,
		"longform_notetweets_inline_media_enabled":                                true,
		"responsive_web_grok_image_annotation_enabled":                            true,
		"responsive_web_enhance_cards_enabled":                                    false,
	}
	featsJSON, err := json.Marshal(features)
	if err != nil {
		return nil, err
	}

	fieldToggles := map[string]bool{
		"withArticleRichContentState": true,
		"withArticlePlainText":        false,
		"withGrokAnalyze":             false,
		"withDisallowedReplyControls": false,
	}
	togglesJSON, err := json.Marshal(fieldToggles)
	if err != nil {
		return nil, err
	}

	// Swap in the new query-hash
	apiURI := "https://x.com/i/api/graphql/4b8_JHYvPYebY8PpwfUdIg/TweetDetail"
	values := url.Values{
		"variables":    {string(varsJSON)},
		"features":     {string(featsJSON)},
		"fieldToggles": {string(togglesJSON)},
	}

	res, err := a.handleGetRequest(apiURI, values)
	if err != nil {
		return nil, err
	}

	var detail *StatusTweet
	if err = a.mapAPIResponse(res, &detail); err != nil {
		return nil, err
	}

	return detail, nil
}
