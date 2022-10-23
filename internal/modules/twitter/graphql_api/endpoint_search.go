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

type SearchTimeline struct {
	GlobalObjects struct {
		Tweets map[string]struct {
			ID               json.Number  `json:"id_str"`
			UserID           json.Number  `json:"user_id"`
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
		} `json:"tweets"`
		Users map[string]struct {
			ID         json.Number `json:"id_str"`
			Name       string      `json:"name"`
			ScreenName string      `json:"screen_name"`
		} `json:"users"`
	} `json:"globalObjects"`
	Timeline struct {
		Instructions []struct {
			AddEntries *struct {
				Entries        []SearchEntry `json:"entries"`
				EntryToReplace string        `json:"entryIdToReplace"`
				Entry          SearchEntry   `json:"entry"`
			} `json:"addEntries"`
			ReplaceEntry *struct {
				Entries        []SearchEntry `json:"entries"`
				EntryToReplace string        `json:"entryIdToReplace"`
				Entry          SearchEntry   `json:"entry"`
			} `json:"replaceEntry"`
		} `json:"instructions"`
	} `json:"timeline"`
}

// GetUserFromGlobalObjectsByUserID returns a User reference if it exists in the global objects of the search
func (t *SearchTimeline) GetUserFromGlobalObjectsByUserID(userID string) *User {
	for _, userEntry := range t.GlobalObjects.Users {
		if userEntry.ID.String() == userID {
			return &User{
				// we don't have the ID in this context, only the Rest ID
				ID:     "",
				RestID: json.Number(userID),
				Legacy: struct {
					Name       string `json:"name"`
					ScreenName string `json:"screen_name"`
				}{
					Name:       userEntry.Name,
					ScreenName: userEntry.ScreenName,
				},
			}
		}
	}

	return nil
}

func (t *SearchTimeline) GetTweetFromGlobalObjectsByTweetID(tweetID string) *Tweet {
	for _, tweetEntry := range t.GlobalObjects.Tweets {
		if tweetEntry.ID.String() == tweetID {
			tweet := Tweet{}
			tweet.EntryID = tweetID

			// tweet specific strings we can't copy from our current context
			tweet.Content.EntryType = "TimelineTimelineItem"
			tweet.Content.ItemContent.ItemType = "TimelineTweet"
			tweet.Content.ItemContent.TweetDisplayType = "Tweet"

			tweet.Content.ItemContent.TweetResults.Result.RestID = json.Number(tweetID)
			tweet.Content.ItemContent.TweetResults.Result.Legacy.CreatedAt = tweetEntry.CreatedAt
			tweet.Content.ItemContent.TweetResults.Result.Legacy.ExtendedEntities = tweetEntry.ExtendedEntities
			tweet.Content.ItemContent.TweetResults.Result.Core.UserResults.Result = t.GetUserFromGlobalObjectsByUserID(tweetEntry.UserID.String())

			return &tweet
		}
	}

	return nil
}

func (t *SearchTimeline) TweetEntries(_ ...string) (tweets []*Tweet) {
	for _, instructions := range t.Timeline.Instructions {
		if instructions.AddEntries != nil {
			for _, entry := range instructions.AddEntries.Entries {
				if entry.Content.Item != nil && entry.Content.Item.Content.Tweet.DisplayType == "Tweet" {
					tweets = append(tweets, t.GetTweetFromGlobalObjectsByTweetID(entry.Content.Item.Content.Tweet.ID.String()))
				}
			}
		}
	}

	return tweets
}

func (t *SearchTimeline) BottomCursor() string {
	for _, instruction := range t.Timeline.Instructions {
		if instruction.AddEntries != nil {
			for _, entry := range instruction.AddEntries.Entries {
				if entry.Content.Operation != nil && entry.Content.Operation.Cursor.CursorType == "Bottom" {
					return entry.Content.Operation.Cursor.Value
				}
			}
		}

		// page 2 and onward replace the cursor entry, only page 1 returns it in "addEntries"
		if instruction.ReplaceEntry != nil {
			if instruction.ReplaceEntry.Entry.Content.Operation.Cursor.CursorType == "Bottom" {
				return instruction.ReplaceEntry.Entry.Content.Operation.Cursor.Value
			}
		}
	}

	return ""
}

func (a *TwitterGraphQlAPI) Search(
	authorName string, untilDate time.Time, cursor string,
) (TimelineInterface, error) {
	a.applyRateLimit()

	apiURI := "https://twitter.com/i/api/2/search/adaptive.json"
	values := url.Values{
		"include_profile_interstitial_type":    {"1"},
		"include_blocking":                     {"1"},
		"include_blocked_by":                   {"1"},
		"include_followed_by":                  {"1"},
		"include_want_retweets":                {"1"},
		"include_mute_edge":                    {"1"},
		"include_can_dm":                       {"1"},
		"include_can_media_tag":                {"1"},
		"include_ext_has_nft_avatar":           {"1"},
		"skip_status":                          {"1"},
		"cards_platform":                       {"Web-12"},
		"include_cards":                        {"1"},
		"include_ext_alt_text":                 {"true"},
		"include_ext_limited_action_results":   {"false"},
		"include_quote_count":                  {"true"},
		"include_reply_count":                  {"1"},
		"tweet_mode":                           {"extended"},
		"include_ext_collab_control":           {"true"},
		"include_entities":                     {"true"},
		"include_user_entities":                {"true"},
		"include_ext_media_color":              {"true"},
		"include_ext_media_availability":       {"true"},
		"include_ext_sensitive_media_warning":  {"true"},
		"include_ext_trusted_friends_metadata": {"true"},
		"send_error_codes":                     {"true"},
		"simple_quoted_tweet":                  {"true"},
		"q":                                    {fmt.Sprintf("(from:%s) until:%s filter:links", authorName, untilDate.Format("2006-01-02"))},
		"count":                                {"20"},
		"query_source":                         {"typed_query"},
		"pc":                                   {"1"},
		"spelling_corrections":                 {"1"},
		"include_ext_edit_control":             {"true"},
		"ext":                                  {"mediaStats,highlightedLabel,hasNftAvatar,voiceInfo,enrichments,superFollowMetadata,unmentionInfo,editControl,collab_control,vibe"},
	}

	if cursor != "" {
		values.Set("cursor", cursor)
	}

	res, err := a.apiGET(apiURI, values)
	if err != nil {
		return nil, err
	}

	var timeline *SearchTimeline
	err = a.mapAPIResponse(res, &timeline)

	return timeline, err
}
