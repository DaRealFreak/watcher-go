package graphql_api

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestTwitterGraphQlAPI_Search(t *testing.T) {
	untilDate := time.Date(2014, time.Month(12), 16, 0, 0, 0, 0, time.UTC)
	res, err := twitterAPI.Search("dareaifreak", untilDate, "")
	assert.New(t).NoError(err)
	assert.New(t).Equal(1, len(res.TweetEntries()))
}
