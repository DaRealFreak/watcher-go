package graphql_api

import (
	"github.com/DaRealFreak/watcher-go/internal/modules/twitter/twitter_settings"
	http "github.com/bogdanfinn/fhttp"
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
	twitterAPI = NewTwitterAPI("twitter API", twitter_settings.TwitterSettings{
		UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:138.0) Gecko/20100101 Firefox/138.0",
	}, nil)
	twitterAPI.SetCookies([]*http.Cookie{
		{
			Name:   "auth_token",
			Value:  os.Getenv("TWITTER_AUTH_COOKIE"),
			MaxAge: 0,
		},
	})

	if os.Getenv("TWITTER_SESSION_COOKIE") != "" {
		twitterAPI.SetCookies([]*http.Cookie{
			{
				Name:   "ct0",
				Value:  os.Getenv("TWITTER_SESSION_COOKIE"),
				MaxAge: 0,
			},
		})
	}

	err := twitterAPI.InitializeSession()
	if err != nil {
		panic(err)
	}

	// run the unit tests
	os.Exit(m.Run())
}

func TestNewTwitterAPI(t *testing.T) {
	res, err := twitterAPI.UserTimelineV2("2923538614", "")
	assert.New(t).NoError(err)
	assert.GreaterOrEqual(t, len(res.TweetEntries("2923538614")), 1)
}
