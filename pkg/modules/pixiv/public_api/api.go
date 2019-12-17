// Package publicapi handles the default API functionality reverse engineered from the public API
package publicapi

import (
	"github.com/DaRealFreak/watcher-go/pkg/models"
	pixivapi "github.com/DaRealFreak/watcher-go/pkg/modules/pixiv/pixiv_api"
)

// PublicAPI is the implementation of the public API
type PublicAPI struct {
	pixivapi.PixivAPI
}

// NewPublicAPI initializes the public API and handles the whole OAuth2 and round tripper procedures
func NewPublicAPI(moduleKey string, account *models.Account) (*PublicAPI, error) {
	pixivAPI, err := pixivapi.NewPixivAPI(moduleKey, account, "http://spapi.pixiv.net/")
	if err != nil {
		return nil, err
	}

	publicAPI := &PublicAPI{
		pixivAPI,
	}

	return publicAPI, nil
}
