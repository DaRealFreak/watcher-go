package graphql_api

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTwitterGraphQlAPI_StatusTweet(t *testing.T) {
	res, err := twitterAPI.StatusTweet("1734483472612462626")
	assert.New(t).NoError(err)
	assert.New(t).GreaterOrEqual(1, len(res.TweetEntries()))
}

// TestStatusTweetTweetEntriesSkipsNonTweetEntries guards the
// `parseStatusGraphQLApi` panic: TweetDetail responses include cursor and
// conversation-module entries whose content has no tweet payload, and the old
// implementation nil-deref'd on Result.RestID for those entries.
func TestStatusTweetTweetEntriesSkipsNonTweetEntries(t *testing.T) {
	realTweet := TweetEntry{
		EntryID: "tweet-1734483472612462626",
		Content: TweetContent{
			EntryType: "TimelineTimelineItem",
			ItemContent: TweetInner{
				ItemType: "TimelineTweet",
				TweetResults: struct {
					Result *SingleTweet `json:"result"`
				}{
					Result: &SingleTweet{RestID: json.Number("1734483472612462626")},
				},
			},
		},
	}

	cursorEntry := TweetEntry{
		EntryID: "cursor-bottom-xyz",
		Content: TweetContent{EntryType: "TimelineTimelineCursor"},
	}

	nilResultEntry := TweetEntry{
		EntryID: "tweet-deleted",
		Content: TweetContent{EntryType: "TimelineTimelineItem"},
	}

	status := &StatusTweet{}
	status.Data.Thread.Instructions = []Instruction[TweetEntry]{
		{
			Type:    "TimelineAddEntries",
			Entries: []TweetEntry{cursorEntry, nilResultEntry, realTweet},
		},
	}

	var tweets []*Tweet
	assert.NotPanics(t, func() {
		tweets = status.TweetEntries()
	})

	assert.Len(t, tweets, 1)
	assert.Equal(t, "tweet-1734483472612462626", tweets[0].EntryID)
}
