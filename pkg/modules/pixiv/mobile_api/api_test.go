package mobileapi

import (
	"os"
	"testing"

	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/stretchr/testify/assert"
)

func getTestMobileAPI() *MobileAPI {
	testAccount := &models.Account{
		Username: os.Getenv("PIXIV_USER"),
		Password: os.Getenv("PIXIV_PASS"),
	}

	mobileAPI := NewMobileAPI("pixiv Mobile API", testAccount)
	if err := mobileAPI.AddRoundTrippers(); err != nil {
		return nil
	}

	return mobileAPI
}

func TestLogin(t *testing.T) {
	testAccount := &models.Account{
		Username: os.Getenv("PIXIV_USER"),
		Password: os.Getenv("PIXIV_PASS"),
	}

	mobileAPI := NewMobileAPI("pixiv Mobile API", testAccount)
	assert.New(t).NotNil(mobileAPI)
}
