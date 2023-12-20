package graphql_api

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTwitterGraphQlAPI_StatusTweet(t *testing.T) {
	res, err := twitterAPI.StatusTweet("1734483472612462626")
	assert.New(t).NoError(err)
	assert.New(t).GreaterOrEqual(1, len(res.TweetEntries()))
}
