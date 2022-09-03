package api

import (
	"io"
	"net/url"
	"os"
	"testing"
	"time"

	cloudflarebp "github.com/DaRealFreak/cloudflare-bp-go"
	"github.com/DaRealFreak/watcher-go/internal/models"
	implicitoauth2 "github.com/DaRealFreak/watcher-go/pkg/oauth2"
	"github.com/stretchr/testify/assert"
)

// nolint: gochecknoglobals
var daAPI *DeviantartAPI

// TestMain is the constructor for the test functions to use a shared API instance
// to prevent multiple logins for every test
func TestMain(m *testing.M) {
	testAccount := &models.Account{
		Username: os.Getenv("DEVIANTART_USER"),
		Password: os.Getenv("DEVIANTART_PASS"),
	}

	// initialize the shared API instance
	daAPI = NewDeviantartAPI("deviantart API", testAccount)
	daAPI.AddRoundTrippers("")

	// run the unit tests
	os.Exit(m.Run())
}

func TestNewDeviantartAPI(t *testing.T) {
	daAPI.useConsoleExploit = false

	res, err := daAPI.request("GET", "/placebo", url.Values{})
	assert.New(t).NoError(err)
	assert.New(t).Equal(200, res.StatusCode)

	contentAPI, err := io.ReadAll(res.Body)
	assert.New(t).NoError(err)

	// toggle console exploit, we also require the first OAuth2 process to have succeeded
	// since we require the user information cookie which is set on a successful login
	daAPI.useConsoleExploit = true

	res, err = daAPI.request("GET", "/placebo", url.Values{})
	assert.New(t).NoError(err)
	assert.New(t).Equal(200, res.StatusCode)

	contentConsoleExploit, err := io.ReadAll(res.Body)
	assert.New(t).NoError(err)

	assert.New(t).Equal(contentAPI, contentConsoleExploit)
}

func TestNewDeviantartAPIExpiredToken(t *testing.T) {
	testAccount := &models.Account{
		Username: os.Getenv("DEVIANTART_USER"),
		Password: os.Getenv("DEVIANTART_PASS"),
	}

	// initialize the shared API instance
	api := NewDeviantartAPI("token expiration test", testAccount)

	client := api.UserSession.GetClient()
	// apply CloudFlare bypass
	client.Transport = cloudflarebp.AddCloudFlareByPass(client.Transport)
	client.Transport = api.setDeviantArtHeaders(client.Transport)

	ts := &implicitoauth2.ImplicitGrantTokenSource{
		Grant: NewImplicitGrantDeviantart(api.OAuth2Config, client, api.account),
	}

	token, err := ts.Token()
	assert.New(t).NoError(err)
	assert.New(t).Equal("bearer", token.TokenType)

	// expire token to force a refresh
	token.Expiry = time.Now().Add(-1 * time.Minute)

	token, err = ts.Token()
	assert.New(t).NoError(err)
	assert.New(t).Equal("bearer", token.TokenType)
}
