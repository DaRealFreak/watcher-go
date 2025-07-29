package graphql_api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
)

type TimelineInterface interface {
	TweetEntries(userIDs ...string) (tweets []*Tweet)
	TombstoneEntries() (tweets []*Tweet)
	BottomCursor() string
}

// TweetData returns the actual tweet entry
func (s *SingleTweet) TweetData() *SingleTweet {
	// "__typename": "TweetWithVisibilityResults" wraps the actual tweet data into another tweet object
	// unsure if this is the case for more types currently
	if s == nil || s.Tweet == nil {
		return s
	} else {
		return s.Tweet
	}
}

type TwitterUserTimeline struct {
	Data struct {
		User struct {
			Result struct {
				Timeline struct {
					Timeline *Timeline `json:"timeline"`
				} `json:"timeline"`
			} `json:"result"`
		} `json:"user"`
	} `json:"data"`
}

// TweetEntries returns all tweet entries from the entries in the timeline response (it also returns cursor entries)
func (t *TwitterUserTimeline) TweetEntries(userIDs ...string) (tweets []*Tweet) {
	return t.Data.User.Result.Timeline.Timeline.TweetEntries(userIDs...)
}

func (t *TwitterUserTimeline) TombstoneEntries() (tweets []*Tweet) {
	return t.Data.User.Result.Timeline.Timeline.TombstoneEntries()
}

// BottomCursor checks for the next cursor in the timeline response
func (t *TwitterUserTimeline) BottomCursor() string {
	return t.Data.User.Result.Timeline.Timeline.BottomCursor()
}

// DownloadItems returns the normalized DownloadQueueItems from the tweet objects
func (tw *Tweet) DownloadItems() (items []*models.DownloadQueueItem) {
	for _, mediaEntry := range tw.Item.ItemContent.TweetResults.Result.TweetData().Legacy.ExtendedEntities.Media {
		if mediaEntry.Type == "video" || mediaEntry.Type == "animated_gif" {
			highestBitRateIndex := 0
			highestBitRate := 0
			for bitRateIndex, variant := range mediaEntry.VideoInfo.Variants {
				if variant.Bitrate >= highestBitRate {
					highestBitRateIndex = bitRateIndex
					highestBitRate = variant.Bitrate
				}
			}

			items = append(items, &models.DownloadQueueItem{
				ItemID:      tw.Item.ItemContent.TweetResults.Result.TweetData().RestID.String(),
				DownloadTag: tw.Item.ItemContent.TweetResults.Result.TweetData().Core.UserResults.Result.Core.ScreenName,
				FileName: fmt.Sprintf(
					"%s_%s_%d_%s",
					tw.Item.ItemContent.TweetResults.Result.TweetData().RestID.String(),
					tw.Item.ItemContent.TweetResults.Result.TweetData().Core.UserResults.Result.RestID.String(),
					len(items)+1,
					fp.GetFileName(mediaEntry.VideoInfo.Variants[highestBitRateIndex].URL),
				),
				FileURI: mediaEntry.VideoInfo.Variants[highestBitRateIndex].URL,
			})
		} else {
			fileType := strings.TrimLeft(fp.GetFileExtension(mediaEntry.MediaURL), ".")
			items = append(items, &models.DownloadQueueItem{
				ItemID:      tw.Item.ItemContent.TweetResults.Result.TweetData().RestID.String(),
				DownloadTag: tw.Item.ItemContent.TweetResults.Result.TweetData().Core.UserResults.Result.Core.ScreenName,
				FileName: fmt.Sprintf(
					"%s_%s_%d_%s",
					tw.Item.ItemContent.TweetResults.Result.TweetData().RestID.String(),
					tw.Item.ItemContent.TweetResults.Result.TweetData().Core.UserResults.Result.RestID.String(),
					len(items)+1,
					fp.GetFileName(mediaEntry.MediaURL),
				),
				FileURI: mediaEntry.MediaURL + "?format=" + fileType + "&name=orig",
			})
		}
	}

	return items
}

func (a *TwitterGraphQlAPI) UserTimelineV2(
	userId string,
	cursor string,
) (TimelineInterface, error) {
	a.applyRateLimit()

	variables := map[string]interface{}{
		"userId":                 userId,
		"count":                  25,
		"includePromotedContent": false,
		"withClientEventToken":   false,
		"withBirdwatchNotes":     false,
		"withVoice":              true,
	}

	if cursor != "" {
		variables["cursor"] = cursor
	}

	varsJSON, err := json.Marshal(variables)
	if err != nil {
		return nil, err
	}

	// Features have changed: match the console-observed set
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
		"withArticlePlainText": false,
	}
	togglesJSON, err := json.Marshal(fieldToggles)
	if err != nil {
		return nil, err
	}

	// Swap in the new query-hash and path
	apiURI := "https://x.com/i/api/graphql/KDdlIFeZgikR2366ZRn0sw/UserMedia"
	values := url.Values{
		"variables":    {string(varsJSON)},
		"features":     {string(featsJSON)},
		"fieldToggles": {string(togglesJSON)},
	}

	res, err := a.handleGetRequest(apiURI, values)
	if err != nil {
		return nil, err
	}

	var timeline *TwitterUserTimeline
	if err = a.mapAPIResponse(res, &timeline); err != nil {
		return nil, err
	}

	return timeline, nil
}

func (a *TwitterGraphQlAPI) UserByUsername(username string) (*UserInformation, error) {
	a.applyRateLimit()

	variables := map[string]interface{}{
		"screen_name": username,
	}
	varsJSON, err := json.Marshal(variables)
	if err != nil {
		return nil, err
	}

	features := map[string]interface{}{
		"responsive_web_grok_bio_auto_translation_is_enabled":               false,
		"hidden_profile_subscriptions_enabled":                              true,
		"payments_enabled":                                                  false,
		"profile_label_improvements_pcf_label_in_post_enabled":              true,
		"rweb_tipjar_consumption_enabled":                                   true,
		"verified_phone_label_enabled":                                      false,
		"subscriptions_verification_info_is_identity_verified_enabled":      true,
		"subscriptions_verification_info_verified_since_enabled":            true,
		"highlights_tweets_tab_ui_enabled":                                  true,
		"responsive_web_twitter_article_notes_tab_enabled":                  true,
		"subscriptions_feature_can_gift_premium":                            true,
		"creator_subscriptions_tweet_preview_api_enabled":                   true,
		"responsive_web_graphql_skip_user_profile_image_extensions_enabled": false,
		"responsive_web_graphql_timeline_navigation_enabled":                true,
	}
	featsJSON, err := json.Marshal(features)
	if err != nil {
		return nil, err
	}

	fieldToggles := map[string]bool{
		"withAuxiliaryUserLabels": true,
	}
	togglesJSON, err := json.Marshal(fieldToggles)
	if err != nil {
		return nil, err
	}

	apiURI := "https://x.com/i/api/graphql/jUKA--0QkqGIFhmfRZdWrQ/UserByScreenName"
	values := url.Values{
		"variables":    {string(varsJSON)},
		"features":     {string(featsJSON)},
		"fieldToggles": {string(togglesJSON)},
	}

	res, err := a.handleGetRequest(apiURI, values)
	if err != nil {
		return nil, err
	}

	var userInformation *UserInformation
	if err = a.mapAPIResponse(res, &userInformation); err != nil {
		return nil, err
	}

	return userInformation, nil
}

func (a *TwitterGraphQlAPI) FollowUser(userId string) error {
	a.applyRateLimit()

	form := url.Values{
		"include_profile_interstitial_type": {"1"},
		"include_blocking":                  {"1"},
		"include_blocked_by":                {"1"},
		"include_followed_by":               {"1"},
		"include_want_retweets":             {"1"},
		"include_mute_edge":                 {"1"},
		"include_can_dm":                    {"1"},
		"include_can_media_tag":             {"1"},
		"include_ext_is_blue_verified":      {"1"},
		"include_ext_verified_type":         {"1"},
		"include_ext_profile_image_shape":   {"1"},
		"skip_status":                       {"1"},
		"user_id":                           {userId},
	}

	apiURI := "https://x.com/i/api/1.1/friendships/create.json"
	_, err := a.handlePostRequest(apiURI, form)
	if err != nil {
		return err
	}

	return nil
}
