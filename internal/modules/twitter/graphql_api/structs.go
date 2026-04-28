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
	Tweet     *SingleTweet `json:"tweet"`
	RestID    json.Number  `json:"rest_id"`
	Tombstone *struct {
		Text struct {
			Text string `json:"text"`
			RTL  bool   `json:"rtl"`
		} `json:"text"`
	} `json:"tombstone"`
	Core struct {
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
						URL     string `json:"url"`
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

type UserURL struct {
	DisplayURL  string `json:"display_url"`
	ExpandedURL string `json:"expanded_url"`
	URL         string `json:"url"`
	Indices     []int  `json:"indices,omitempty"`
}

type UserEntities struct {
	Description struct {
		URLs []UserURL `json:"urls"`
	} `json:"description"`
	URL struct {
		URLs []UserURL `json:"urls"`
	} `json:"url"`
}

type UserBirthdate struct {
	Day            int    `json:"day"`
	Month          int    `json:"month"`
	Year           int    `json:"year,omitempty"`
	Visibility     string `json:"visibility,omitempty"`
	YearVisibility string `json:"year_visibility,omitempty"`
}

type User struct {
	ID     string      `json:"id"`
	RestID json.Number `json:"rest_id"`
	Avatar struct {
		ImageURL string `json:"image_url"`
	} `json:"avatar"`
	Core struct {
		Name       string `json:"name"`
		ScreenName string `json:"screen_name"`
		CreatedAt  string `json:"created_at,omitempty"`
	} `json:"core"`
	Privacy struct {
		Protected bool `json:"protected"`
	} `json:"privacy"`
	Location struct {
		Location string `json:"location"`
	} `json:"location"`
	LegacyExtendedProfile struct {
		Birthdate *UserBirthdate `json:"birthdate,omitempty"`
	} `json:"legacy_extended_profile"`
	RelationshipPerspectives struct {
		Following bool `json:"following"`
	} `json:"relationship_perspectives"`
	Legacy struct {
		FollowRequestSent *bool        `json:"follow_request_sent"`
		Description       string       `json:"description,omitempty"`
		URL               string       `json:"url,omitempty"`
		Entities          UserEntities `json:"entities"`
		ProfileBannerURL  string       `json:"profile_banner_url,omitempty"`
		FollowersCount    int          `json:"followers_count,omitempty"`
		FriendsCount      int          `json:"friends_count,omitempty"`
		MediaCount        int          `json:"media_count,omitempty"`
		StatusesCount     int          `json:"statuses_count,omitempty"`
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
