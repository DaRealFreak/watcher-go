package publicapi

import (
	"os"
	"testing"

	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/stretchr/testify/assert"
)

func getTestPublicAPI() *PublicAPI {
	testAccount := &models.Account{
		Username: os.Getenv("PIXIV_USER"),
		Password: os.Getenv("PIXIV_PASS"),
	}

	publicAPI := NewPublicAPI("pixiv Mobile API", testAccount)
	if err := publicAPI.AddRoundTrippers(); err != nil {
		return nil
	}

	return publicAPI
}

func TestLogin(t *testing.T) {
	testAccount := &models.Account{
		Username: os.Getenv("PIXIV_USER"),
		Password: os.Getenv("PIXIV_PASS"),
	}

	mobileAPI := NewPublicAPI("pixiv Mobile API", testAccount)
	assert.New(t).NotNil(mobileAPI)
}
