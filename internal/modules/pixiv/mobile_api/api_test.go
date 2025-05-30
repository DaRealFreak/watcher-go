package mobileapi

import (
	"os"
	"testing"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/stretchr/testify/assert"
)

// nolint: gochecknoglobals
var mobileAPI *MobileAPI

// TestMain is the constructor for the API functions creating a shared instance for the API
// to prevent multiple consecutive logins
func TestMain(m *testing.M) {
	testAccount := &models.OAuthClient{
		ClientID:     os.Getenv("PIXIV_CLIENT_ID"),
		ClientSecret: os.Getenv("PIXIV_CLIENT_SECRET"),
		AccessToken:  os.Getenv("PIXIV_ACCESS_TOKEN"),
		RefreshToken: os.Getenv("PIXIV_REFRESH_TOKEN"),
	}

	mobileAPI = NewMobileAPI("pixiv Mobile API", testAccount)
	if err := mobileAPI.ConfigureTokenSource(); err != nil {
		os.Exit(1)
	}

	// run the unit tests
	os.Exit(m.Run())
}

func TestLogin(t *testing.T) {
	testAccount := &models.OAuthClient{
		ClientID:     os.Getenv("PIXIV_CLIENT_ID"),
		ClientSecret: os.Getenv("PIXIV_CLIENT_SECRET"),
		AccessToken:  os.Getenv("PIXIV_ACCESS_TOKEN"),
		RefreshToken: os.Getenv("PIXIV_REFRESH_TOKEN"),
	}

	mobileAPI := NewMobileAPI("pixiv Mobile API", testAccount)
	assert.New(t).NotNil(mobileAPI)
	assert.New(t).NoError(mobileAPI.ConfigureTokenSource())
}
