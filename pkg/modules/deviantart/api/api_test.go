package api

import (
	"io/ioutil"
	"net/url"
	"os"
	"testing"

	"github.com/DaRealFreak/watcher-go/pkg/models"
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
	daAPI.AddRoundTrippers()

	// run the unit tests
	os.Exit(m.Run())
}

func TestNewDeviantartAPI(t *testing.T) {
	res, err := daAPI.request("GET", "/placebo", url.Values{})
	assert.New(t).NoError(err)
	assert.New(t).Equal(200, res.StatusCode)

	contentAPI, err := ioutil.ReadAll(res.Body)
	assert.New(t).NoError(err)

	// toggle console exploit, we also require the first OAuth2 process to have succeeded
	// since we require the user information cookie which is set on a successful login
	daAPI.useConsoleExploit = true

	res, err = daAPI.request("GET", "/placebo", url.Values{})
	assert.New(t).NoError(err)
	assert.New(t).Equal(200, res.StatusCode)

	contentConsoleExploit, err := ioutil.ReadAll(res.Body)
	assert.New(t).NoError(err)

	assert.New(t).Equal(contentAPI, contentConsoleExploit)
}
