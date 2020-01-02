package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTwitterAPI_UserTimeline(t *testing.T) {
	tweets, err := twitterAPI.UserTimeline(
		"DaReaIFreak", "", "", MaxTweetsPerRequest, true,
	)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(tweets)
}
