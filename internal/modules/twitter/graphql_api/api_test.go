package graphql_api

import (
	"net/http"
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// nolint: gochecknoglobals
var twitterAPI *TwitterGraphQlAPI

// TestMain is the constructor for the test functions to use a shared API instance
// to prevent multiple logins for every test
func TestMain(m *testing.M) {
	// initialize the shared API instance
	twitterAPI = NewTwitterAPI("twitter API")
	twitterAPI.AddRoundTrippers()

	requestUrl, _ := url.Parse("https://twitter.com/")
	twitterAPI.Session.GetClient().Jar.SetCookies(
		requestUrl,
		[]*http.Cookie{
			{
				Name:   "auth_token",
				Value:  os.Getenv("TWITTER_AUTH_COOKIE"),
				MaxAge: 0,
			},
		},
	)

	// run the unit tests
	os.Exit(m.Run())
}

func TestNewTwitterAPI(t *testing.T) {
	res, err := twitterAPI.UserTimelineV2("2923538614", "")
	assert.New(t).NoError(err)
	assert.GreaterOrEqual(t, len(res.TweetEntries("2923538614")), 4)
}
