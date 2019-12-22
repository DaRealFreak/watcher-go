package api

import (
	"io/ioutil"
	"net/url"
	"os"
	"testing"

	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/stretchr/testify/assert"
)

func TestNewDeviantartAPI(t *testing.T) {
	testAccount := &models.Account{
		Username: os.Getenv("DEVIANTART_USER"),
		Password: os.Getenv("DEVIANTART_PASS"),
	}

	daAPI := NewDeviantartAPI("deviantart API", testAccount)
	daAPI.AddRoundTrippers()

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
