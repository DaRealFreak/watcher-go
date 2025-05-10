package graphql_api

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestTwitterGraphQlAPI_Search(t *testing.T) {
	untilDate := time.Date(2022, time.Month(1), 2, 0, 0, 0, 0, time.UTC)
	res, err := twitterAPI.Search("anybrody", untilDate, "")
	assert.New(t).NoError(err)
	assert.New(t).Equal(20, len(res.TweetEntries()))
}
