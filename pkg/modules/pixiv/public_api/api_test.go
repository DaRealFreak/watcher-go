package publicapi

import (
	"os"
	"testing"

	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/stretchr/testify/assert"
)

func getTestMobileAPI() *PublicAPI {
	testAccount := models.Account{
		Username: os.Getenv("PIXIV_USER"),
		Password: os.Getenv("PIXIV_PASS"),
	}

	mobileAPI, _ := NewPublicAPI("pixiv Mobile API", testAccount)

	return mobileAPI
}

func TestLogin(t *testing.T) {
	testAccount := models.Account{
		Username: os.Getenv("PIXIV_USER"),
		Password: os.Getenv("PIXIV_PASS"),
	}

	mobileAPI, err := NewPublicAPI("pixiv Mobile API", testAccount)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(mobileAPI)
}
