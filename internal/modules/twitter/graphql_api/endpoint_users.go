package graphql_api

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
)

type Tweet struct {
	EntryID string `json:"entryId"`
	Content struct {
		EntryType   string `json:"entryType"`
		Value       string `json:"value"`
		CursorType  string `json:"cursorType"`
		ItemContent struct {
			ItemType     string `json:"itemType"`
			TweetResults struct {
				Result struct {
					RestID json.Number `json:"rest_id"`
					Core   struct {
						UserResults struct {
							Result User `json:"result"`
						} `json:"user_results"`
					} `json:"core"`
					Legacy struct {
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
				} `json:"result"`
			} `json:"tweet_results"`
			TweetDisplayType string `json:"tweetDisplayType"`
		} `json:"itemContent"`
	} `json:"content"`
}

type StatusTweet struct {
	Data struct {
		ThreadedConversationWithInjectionsV2 struct {
			Instructions []struct {
				Type    string   `json:"type"`
				Entries []*Tweet `json:"entries"`
			} `json:"instructions"`
		} `json:"threaded_conversation_with_injections_v2"`
	} `json:"data"`
}

type Timeline struct {
	Data struct {
		User struct {
			Result struct {
				TimelineV2 struct {
					Timeline struct {
						Instructions []struct {
							Type    string   `json:"type"`
							Entries []*Tweet `json:"entries"`
						} `json:"instructions"`
					} `json:"timeline"`
				} `json:"timeline_v2"`
			} `json:"result"`
		} `json:"user"`
	} `json:"data"`
}

// TweetEntries returns all tweet entries from the entries in the timeline response (it also returns cursor entries)
func (t *Timeline) TweetEntries(userIDs ...string) (tweets []*Tweet) {
	for _, instruction := range t.Data.User.Result.TimelineV2.Timeline.Instructions {
		if instruction.Type != "TimelineAddEntries" {
			continue
		}

		for _, entry := range instruction.Entries {
			if entry.Content.EntryType != "TimelineTimelineItem" {
				continue
			}

			if len(userIDs) != 0 {
				inAllowedUsers := false
				for _, userID := range userIDs {
					if userID == entry.Content.ItemContent.TweetResults.Result.Core.UserResults.Result.RestID.String() {
						inAllowedUsers = true
						break
					}
				}

				// not in allowed users, skip entry (most likely advertisement entries)
				if !inAllowedUsers {
					continue
				}
			}

			tweets = append(tweets, entry)
		}
	}

	return tweets
}

// DownloadItems returns the normalized DownloadQueueItems from the tweet objects
func (tw *Tweet) DownloadItems() (items []*models.DownloadQueueItem) {
	for _, mediaEntry := range tw.Content.ItemContent.TweetResults.Result.Legacy.ExtendedEntities.Media {
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
				ItemID:      tw.Content.ItemContent.TweetResults.Result.RestID.String(),
				DownloadTag: tw.Content.ItemContent.TweetResults.Result.Core.UserResults.Result.Legacy.ScreenName,
				FileName: fmt.Sprintf(
					"%s_%s_%d_%s",
					tw.Content.ItemContent.TweetResults.Result.RestID.String(),
					tw.Content.ItemContent.TweetResults.Result.Core.UserResults.Result.RestID.String(),
					len(items)+1,
					fp.GetFileName(mediaEntry.VideoInfo.Variants[highestBitRateIndex].URL),
				),
				FileURI: mediaEntry.VideoInfo.Variants[highestBitRateIndex].URL,
			})
		} else {
			items = append(items, &models.DownloadQueueItem{
				ItemID:      tw.Content.ItemContent.TweetResults.Result.RestID.String(),
				DownloadTag: tw.Content.ItemContent.TweetResults.Result.Core.UserResults.Result.Legacy.ScreenName,
				FileName: fmt.Sprintf(
					"%s_%s_%d_%s",
					tw.Content.ItemContent.TweetResults.Result.RestID.String(),
					tw.Content.ItemContent.TweetResults.Result.Core.UserResults.Result.RestID.String(),
					len(items)+1,
					fp.GetFileName(mediaEntry.MediaURL),
				),
				FileURI: mediaEntry.MediaURL,
			})
		}
	}

	return items
}

// BottomCursor checks for the next cursor in the timeline response
func (t *Timeline) BottomCursor() string {
	for _, instruction := range t.Data.User.Result.TimelineV2.Timeline.Instructions {
		if instruction.Type != "TimelineAddEntries" {
			continue
		}

		for _, entry := range instruction.Entries {
			if entry.Content.CursorType != "Bottom" {
				continue
			}

			return entry.Content.Value
		}
	}

	return ""
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
) (*Timeline, error) {
	a.applyRateLimit()

	variables := map[string]interface{}{
		"userId":                      userId,
		"count":                       40,
		"includePromotedContent":      false,
		"withSuperFollowsUserFields":  true,
		"withDownvotePerspective":     false,
		"withReactionsMetadata":       false,
		"withReactionsPerspective":    false,
		"withSuperFollowsTweetFields": true,
		"withClientEventToken":        false,
		"withBirdwatchNotes":          false,
		"withVoice":                   true,
		"withV2Timeline":              true,
	}

	if cursor != "" {
		variables["cursor"] = cursor
	}

	variablesJson, _ := json.Marshal(variables)

	featuresJson, _ := json.Marshal(map[string]interface{}{
		"dont_mention_me_view_api_enabled":      true,
		"interactive_text_enabled":              true,
		"responsive_web_uc_gql_enabled":         false,
		"responsive_web_edit_tweet_api_enabled": false,
	})

	apiURI := "https://twitter.com/i/api/graphql/ZnNUqQaF7ZP5sJehbi2u6A/UserMedia"
	values := url.Values{
		"variables": {string(variablesJson)},
		"features":  {string(featuresJson)},
	}

	res, err := a.apiGET(apiURI, values)
	if err != nil {
		return nil, err
	}

	var timeline *Timeline
	err = a.mapAPIResponse(res, &timeline)

	return timeline, err
}

func (a *TwitterGraphQlAPI) UserByUsername(username string) (*UserInformation, error) {
	a.applyRateLimit()

	variablesJson, _ := json.Marshal(map[string]interface{}{
		"screen_name":                username,
		"withSafetyModeUserFields":   true,
		"withSuperFollowsUserFields": true,
	})

	apiURI := "https://twitter.com/i/api/graphql/Bhlf1dYJ3bYCKmLfeEQ31A/UserByScreenName"
	values := url.Values{
		"variables": {string(variablesJson)},
	}

	res, err := a.apiGET(apiURI, values)
	if err != nil {
		return nil, err
	}

	var userInformation *UserInformation
	err = a.mapAPIResponse(res, &userInformation)

	return userInformation, err
}
