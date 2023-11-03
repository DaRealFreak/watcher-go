package graphql_api

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
)

type TimelineInterface interface {
	TweetEntries(userIDs ...string) (tweets []*Tweet)
	BottomCursor() string
}

// TweetData returns the actual tweet entry
func (s *SingleTweet) TweetData() *SingleTweet {
	// "__typename": "TweetWithVisibilityResults" wraps the actual tweet data into another tweet object
	// unsure if this is the case for more types currently
	if s.Tweet != nil {
		return s.Tweet
	} else {
		return s
	}
}

type SingleTweet struct {
	Tweet  *SingleTweet `json:"tweet"`
	RestID json.Number  `json:"rest_id"`
	Core   struct {
		UserResults struct {
			Result *User `json:"result"`
		} `json:"user_results"`
	} `json:"core"`
	Legacy struct {
		CreatedAt        *TwitterTime `json:"created_at"`
		ExtendedEntities struct {
			Media []struct {
				ID        json.Number `json:"id_str"`
				Type      string      `json:"type"`
				MediaURL  string      `json:"media_url_https"`
				VideoInfo *struct {
					Variants []struct {
						Bitrate int    `json:"bitrate"`
						URL     string `json:"URL"`
					} `json:"variants"`
				} `json:"video_info"`
			} `json:"media"`
		} `json:"extended_entities"`
	} `json:"legacy"`
}

type Tweet struct {
	EntryID string `json:"entryId"`
	Content struct {
		EntryType   string `json:"entryType"`
		Value       string `json:"value"`
		CursorType  string `json:"cursorType"`
		ItemContent struct {
			ItemType     string `json:"itemType"`
			TweetResults struct {
				Result *SingleTweet `json:"result"`
			} `json:"tweet_results"`
			TweetDisplayType string `json:"tweetDisplayType"`
		} `json:"itemContent"`
	} `json:"content"`
}

type StatusTweet struct {
	Data struct {
		ThreadedConversationWithInjectionsV2 struct {
			Instructions []TimelineInstruction `json:"instructions"`
		} `json:"threaded_conversation_with_injections_v2"`
	} `json:"data"`
}

type TimelineInstruction struct {
	Type    string   `json:"type"`
	Entries []*Tweet `json:"entries"`
}

type UserTimeline struct {
	Data struct {
		User struct {
			Result struct {
				TimelineV2 struct {
					Timeline *Timeline `json:"timeline"`
				} `json:"timeline_v2"`
			} `json:"result"`
		} `json:"user"`
	} `json:"data"`
}

// TweetEntries returns all tweet entries from the entries in the timeline response (it also returns cursor entries)
func (t *UserTimeline) TweetEntries(userIDs ...string) (tweets []*Tweet) {
	return t.Data.User.Result.TimelineV2.Timeline.TweetEntries(userIDs...)
}

// BottomCursor checks for the next cursor in the timeline response
func (t *UserTimeline) BottomCursor() string {
	return t.Data.User.Result.TimelineV2.Timeline.BottomCursor()
}

// DownloadItems returns the normalized DownloadQueueItems from the tweet objects
func (tw *Tweet) DownloadItems() (items []*models.DownloadQueueItem) {
	for _, mediaEntry := range tw.Content.ItemContent.TweetResults.Result.TweetData().Legacy.ExtendedEntities.Media {
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
				ItemID:      tw.Content.ItemContent.TweetResults.Result.TweetData().RestID.String(),
				DownloadTag: tw.Content.ItemContent.TweetResults.Result.TweetData().Core.UserResults.Result.Legacy.ScreenName,
				FileName: fmt.Sprintf(
					"%s_%s_%d_%s",
					tw.Content.ItemContent.TweetResults.Result.TweetData().RestID.String(),
					tw.Content.ItemContent.TweetResults.Result.TweetData().Core.UserResults.Result.RestID.String(),
					len(items)+1,
					fp.GetFileName(mediaEntry.VideoInfo.Variants[highestBitRateIndex].URL),
				),
				FileURI: mediaEntry.VideoInfo.Variants[highestBitRateIndex].URL,
			})
		} else {
			items = append(items, &models.DownloadQueueItem{
				ItemID:      tw.Content.ItemContent.TweetResults.Result.TweetData().RestID.String(),
				DownloadTag: tw.Content.ItemContent.TweetResults.Result.TweetData().Core.UserResults.Result.Legacy.ScreenName,
				FileName: fmt.Sprintf(
					"%s_%s_%d_%s",
					tw.Content.ItemContent.TweetResults.Result.TweetData().RestID.String(),
					tw.Content.ItemContent.TweetResults.Result.TweetData().Core.UserResults.Result.RestID.String(),
					len(items)+1,
					fp.GetFileName(mediaEntry.MediaURL),
				),
				FileURI: mediaEntry.MediaURL,
			})
		}
	}

	return items
}

type User struct {
	ID     string      `json:"id"`
	RestID json.Number `json:"rest_id"`
	Legacy struct {
		Name       string `json:"name"`
		ScreenName string `json:"screen_name"`
	} `json:"legacy"`
}

type UserInformation struct {
	Data struct {
		User struct {
			Result User `json:"result"`
		} `json:"user"`
	} `json:"data"`
}

func (a *TwitterGraphQlAPI) UserTimelineV2(
	userId string,
	cursor string,
) (TimelineInterface, error) {
	a.applyRateLimit()

	variables := map[string]interface{}{
		"userId":                 userId,
		"count":                  40,
		"includePromotedContent": false,
		"withClientEventToken":   false,
		"withBirdwatchNotes":     false,
		"withVoice":              true,
		"withV2Timeline":         true,
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

	apiURI := "https://twitter.com/i/api/graphql/7_ZP_xN3Bcq1I2QkK5yc2w/UserMedia"
	values := url.Values{
		"variables": {string(variablesJson)},
		"features":  {string(featuresJson)},
	}

	res, err := a.apiGET(apiURI, values)
	if err != nil {
		return nil, err
	}

	var timeline *UserTimeline
	err = a.mapAPIResponse(res, &timeline)

	return timeline, err
}

func (a *TwitterGraphQlAPI) UserByUsername(username string) (*UserInformation, error) {
	a.applyRateLimit()

	variablesJson, _ := json.Marshal(map[string]interface{}{
		"screen_name":              username,
		"withSafetyModeUserFields": true,
	})

	// {"hidden_profile_likes_enabled":true,"hidden_profile_subscriptions_enabled":true,"responsive_web_graphql_exclude_directive_enabled":true,"verified_phone_label_enabled":false,"subscriptions_verification_info_is_identity_verified_enabled":true,"subscriptions_verification_info_verified_since_enabled":true,"highlights_tweets_tab_ui_enabled":true,"creator_subscriptions_tweet_preview_api_enabled":true,"responsive_web_graphql_skip_user_profile_image_extensions_enabled":false,"responsive_web_graphql_timeline_navigation_enabled":true}
	featuresJson, _ := json.Marshal(map[string]interface{}{
		"hidden_profile_likes_enabled":                                      true,
		"hidden_profile_subscriptions_enabled":                              true,
		"responsive_web_graphql_exclude_directive_enabled":                  true,
		"verified_phone_label_enabled":                                      false,
		"subscriptions_verification_info_is_identity_verified_enabled":      true,
		"subscriptions_verification_info_verified_since_enabled":            true,
		"highlights_tweets_tab_ui_enabled":                                  true,
		"creator_subscriptions_tweet_preview_api_enabled":                   true,
		"responsive_web_graphql_skip_user_profile_image_extensions_enabled": false,
		"responsive_web_graphql_timeline_navigation_enabled":                true,
	})

	apiURI := "https://twitter.com/i/api/graphql/G3KGOASz96M-Qu0nwmGXNg/UserByScreenName"
	values := url.Values{
		"variables": {string(variablesJson)},
		"features":  {string(featuresJson)},
	}

	res, err := a.apiGET(apiURI, values)
	if err != nil {
		return nil, err
	}

	var userInformation *UserInformation
	err = a.mapAPIResponse(res, &userInformation)

	return userInformation, err
}
