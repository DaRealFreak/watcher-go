package graphql_api

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTwitterGraphQlAPI_UserTimelineV2(t *testing.T) {
	res, err := twitterAPI.UserTimelineV2("2923538614", "")
	assert.New(t).NoError(err)
	assert.New(t).GreaterOrEqual(4, res.TweetEntries("2923538614"))
}
