package api

import (
	"net/url"
	"os"
	"strconv"
	"testing"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/stretchr/testify/assert"
)

// nolint: gochecknoglobals
var twitterAPI *TwitterAPI

// TestMain is the constructor for the test functions to use a shared API instance
// to prevent multiple logins for every test
func TestMain(m *testing.M) {
	testOAuth2Client := &models.OAuthClient{
		ClientID:     os.Getenv("TWITTER_CLIENT_ID"),
		ClientSecret: os.Getenv("TWITTER_CLIENT_SECRET"),
		AccessToken:  os.Getenv("TWITTER_TOKEN_SOURCE"),
		RefreshToken: os.Getenv("TWITTER_TOKEN_SECRET"),
	}

	// initialize the shared API instance
	twitterAPI = NewTwitterAPI("twitter API", testOAuth2Client)

	// run the unit tests
	os.Exit(m.Run())
}

func TestNewTwitterAPI(t *testing.T) {
	values := url.Values{
		"screen_name": {"DaReaIFreak"},
		"trim_user":   {"1"},
		"count":       {strconv.Itoa(MaxTweetsPerRequest)},
		"include_rts": {"1"},
	}

	apiURI := "https://api.twitter.com/1.1/statuses/user_timeline.json"
	if values.Encode() != "" {
		apiURI += "?" + values.Encode()
	}

	res, err := twitterAPI.Session.Get(apiURI)
	assert.New(t).NoError(err)
	assert.New(t).Equal(200, res.StatusCode)
}
