// Package mobileapi handles the default API functionality reverse engineered from the mobile application
// since the API is not documented or intended to be used outside of the mobile application
package mobileapi

import (
	"github.com/DaRealFreak/watcher-go/pkg/models"
	pixivapi "github.com/DaRealFreak/watcher-go/pkg/modules/pixiv/pixiv_api"
)

// MobileAPI is the implementation of the API used in the mobile applications
type MobileAPI struct {
	pixivapi.PixivAPI
}

// NewMobileAPI initializes the mobile API and handles the whole OAuth2 and round tripper procedures
func NewMobileAPI(moduleKey string, account *models.Account) *MobileAPI {
	return &MobileAPI{
		*pixivapi.NewPixivAPI(moduleKey, account, "https://app-api.pixiv.net/"),
	}
}
