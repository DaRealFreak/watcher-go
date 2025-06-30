package graphql_api

import "encoding/json"

type Entry[C any] struct {
	EntryID string `json:"entryId"`
	Content C      `json:"content"`
}

type Instruction[E any] struct {
	Type    string `json:"type"`
	Entries []E    `json:"entries"`
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

type TweetInner struct {
	ItemType     string `json:"itemType"`
	TweetResults struct {
		Result *SingleTweet `json:"result"`
	} `json:"tweet_results"`
}

type Tweet struct {
	EntryID string `json:"entryId"`
	Item    struct {
		ItemContent struct {
			TweetInner
		} `json:"itemContent"`
	} `json:"item"`
}

type TweetContent struct {
	EntryType   string     `json:"entryType"`
	ItemContent TweetInner `json:"itemContent"`
}

func (tc *TweetContent) GetTweet() *Tweet {
	return &Tweet{
		EntryID: "tweet-" + tc.ItemContent.TweetResults.Result.RestID.String(),
		Item: struct {
			ItemContent struct {
				TweetInner
			} `json:"itemContent"`
		}{
			ItemContent: struct {
				TweetInner
			}{
				TweetInner: tc.ItemContent,
			},
		},
	}
}

type TweetEntry = Entry[TweetContent]

type User struct {
	ID     string      `json:"id"`
	RestID json.Number `json:"rest_id"`
	Core   struct {
		Name       string `json:"name"`
		ScreenName string `json:"screen_name"`
	} `json:"core"`
	Privacy struct {
		Protected bool `json:"protected"`
	} `json:"privacy"`
	RelationshipPerspectives struct {
		Following bool `json:"following"`
	} `json:"relationship_perspectives"`
	Legacy struct {
		FollowRequestSent *bool `json:"follow_request_sent"`
	} `json:"legacy"`
	Message  *string `json:"message"`
	Reason   *string `json:"reason"`
	TypeName *string `json:"__typename"`
}

type UserInformation struct {
	Data struct {
		User struct {
			Result User `json:"result"`
		} `json:"user"`
	} `json:"data"`
}

type SearchEntry = Entry[struct {
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
}]

type WithTweetResult struct {
	ItemType     string `json:"itemType"`
	TweetResults struct {
		Result *SingleTweet `json:"result"`
	} `json:"tweet_results"`
	TweetDisplayType string `json:"tweetDisplayType"`
}

type InstructionEntry = Entry[struct {
	Type    string   `json:"type"`
	Entries []*Tweet `json:"entries"`
}]
