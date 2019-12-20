package api

import (
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
}
