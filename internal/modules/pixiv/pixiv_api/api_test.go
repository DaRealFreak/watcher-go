package pixivapi

import (
	"os"
	"testing"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/stretchr/testify/assert"
)

func getTestPixivAPI() *PixivAPI {
	testAccount := &models.Account{
		Username: os.Getenv("PIXIV_USER"),
		Password: os.Getenv("PIXIV_PASS"),
	}

	pixivAPI := NewPixivAPI("pixiv API", testAccount, "https://app-api.pixiv.net/")
	if err := pixivAPI.AddRoundTrippers(); err != nil {
		return nil
	}

	return pixivAPI
}

func TestNewPixivAPI(t *testing.T) {
	testAccount := &models.Account{
		Username: os.Getenv("PIXIV_USER"),
		Password: os.Getenv("PIXIV_PASS"),
	}

	pixivAPI := NewPixivAPI("pixiv API", testAccount, "https://app-api.pixiv.net/")
	err := pixivAPI.AddRoundTrippers()
	assert.New(t).NoError(err)
	assert.New(t).NotNil(pixivAPI)
}
