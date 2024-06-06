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

	variables := map[string]interface{}{
		"rawQuery":    fmt.Sprintf("(from:%s) until:%s filter:links", authorName, untilDate.Format("2006-01-02")),
		"count":       20,
		"querySource": "typed_query",
		"product":     "Media",
	}

	if cursor != "" {
		variables["cursor"] = cursor
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

	apiURI := "https://x.com/i/api/graphql/lZ0GCEojmtQfiUQa5oJSEw/SearchTimeline"
	values := url.Values{
		"variables": {string(variablesJson)},
		"features":  {string(featuresJson)},
	}

	res, err := a.apiGET(apiURI, values)
	if err != nil {
		return nil, err
	}

	var timelineData *SearchTimelineData
	err = a.mapAPIResponse(res, &timelineData)

	return timelineData, err
}
