package graphql_api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"
)

type SearchEntry struct {
	EntryID string `json:"entryId"`
	Content struct {
		Item *struct {
			Content struct {
				Tweet struct {
					ID          json.Number `json:"id"`
					DisplayType string      `json:"displayType"`
				} `json:"tweet"`
			} `json:"content"`
		} `json:"item"`
		Operation *struct {
			Cursor struct {
				Value      string `json:"value"`
				CursorType string `json:"cursorType"`
			} `json:"cursor"`
		} `json:"operation"`
	} `json:"content"`
}

type SearchTimelineData struct {
	Data struct {
		SearchByRawQuery struct {
			SearchTimeline struct {
				Timeline *Timeline `json:"timeline"`
			} `json:"search_timeline"`
		} `json:"search_by_raw_query"`
	} `json:"data"`
}

func (s *SearchTimelineData) TweetEntries(userIDs ...string) (tweets []*Tweet) {
	return s.Data.SearchByRawQuery.SearchTimeline.Timeline.TweetEntries(userIDs...)
}

func (s *SearchTimelineData) BottomCursor() string {
	return s.Data.SearchByRawQuery.SearchTimeline.Timeline.BottomCursor()
}

func (a *TwitterGraphQlAPI) Search(
	authorName string, untilDate time.Time, cursor string,
) (TimelineInterface, error) {
	a.applyRateLimit()

	rawQuery := fmt.Sprintf("(from:%s) until:%s filter:links",
		authorName, untilDate.Format("2006-01-02"),
	)

	variables := map[string]interface{}{
		"rawQuery":    rawQuery,
		"count":       20,
		"querySource": "typed_query",
		"product":     "Media",
	}

	if cursor != "" {
		variables["cursor"] = cursor
	}

	variablesJson, _ := json.Marshal(variables)
	features := map[string]interface{}{
		"rweb_video_screen_enabled":                                               false,
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

	featuresJson, _ := json.Marshal(features)

	apiURI := "https://x.com/i/api/graphql/VhUd6vHVmLBcw0uX-6jMLA/SearchTimeline"

	// attach as query-params
	values := url.Values{
		"variables": {string(variablesJson)},
		"features":  {string(featuresJson)},
	}

	res, err := a.handleGetRequest(apiURI, values)
	if err != nil {
		return nil, err
	}

	var timelineData *SearchTimelineData
	if err = a.mapAPIResponse(res, &timelineData); err != nil {
		return nil, err
	}

	return timelineData, nil
}
