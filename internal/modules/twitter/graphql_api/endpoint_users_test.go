package graphql_api

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTwitterGraphQlAPI_UserTimelineV2(t *testing.T) {
	res, err := twitterAPI.UserTimelineV2("2923538614", "")
	assert.New(t).NoError(err)
	assert.New(t).GreaterOrEqual(4, len(res.TweetEntries("2923538614")))
}

func TestTwitterGraphQlAPI_UserByUsername(t *testing.T) {
	res, err := twitterAPI.UserByUsername("DaReaiFreak")
	assert.New(t).NoError(err)
	assert.New(t).Equal("2923538614", res.Data.User.Result.RestID.String())
}
