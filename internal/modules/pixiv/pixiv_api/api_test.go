package pixivapi

import (
	"os"
	"testing"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/stretchr/testify/assert"
)

func getTestPixivAPI() *PixivAPI {
	testAccount := &models.OAuthClient{
		ClientID:     os.Getenv("PIXIV_CLIENT_ID"),
		ClientSecret: os.Getenv("PIXIV_CLIENT_SECRET"),
		AccessToken:  os.Getenv("PIXIV_ACCESS_TOKEN"),
		RefreshToken: os.Getenv("PIXIV_REFRESH_TOKEN"),
	}

	pixivAPI := NewPixivAPI("pixiv API", testAccount, "https://app-api.pixiv.net/")
	if err := pixivAPI.ConfigureTokenSource(); err != nil {
		return nil
	}

	return pixivAPI
}

func TestNewPixivAPI(t *testing.T) {
	testAccount := &models.OAuthClient{
		ClientID:     os.Getenv("PIXIV_CLIENT_ID"),
		ClientSecret: os.Getenv("PIXIV_CLIENT_SECRET"),
		AccessToken:  os.Getenv("PIXIV_ACCESS_TOKEN"),
		RefreshToken: os.Getenv("PIXIV_REFRESH_TOKEN"),
	}

	pixivAPI := NewPixivAPI("pixiv API", testAccount, "https://app-api.pixiv.net/")
	err := pixivAPI.ConfigureTokenSource()
	assert.New(t).NoError(err)
	assert.New(t).NotNil(pixivAPI)
}
