package models

import "database/sql"

// DatabaseInterface of used functions from the application to eventually change the underlying library
type DatabaseInterface interface {
	// tracked item storage functionality

	GetTrackedItems(module ModuleInterface, includeCompleted bool) (items []*TrackedItem)
	GetTrackedItemsByDomain(domain string, includeCompleted bool) (items []*TrackedItem)
	GetFirstOrCreateTrackedItem(uri string, subFolder string, module ModuleInterface) *TrackedItem
	GetAllOrCreateTrackedItemIgnoreSubFolder(uri string, module ModuleInterface) (items []*TrackedItem)
	UpdateTrackedItem(trackedItem *TrackedItem, currentItem string)
	ChangeTrackedItemUri(trackedItem *TrackedItem, uri string)
	CreateTrackedItem(uri string, subFolder string, module ModuleInterface)
	ChangeTrackedItemCompleteStatus(trackedItem *TrackedItem, complete bool)
	ChangeTrackedItemSubFolder(trackedItem *TrackedItem, subFolder string)
	ChangeTrackedItemFavoriteStatus(trackedItem *TrackedItem, favorite bool)

	// account storage functionality

	CreateAccount(user string, password string, module ModuleInterface)
	GetFirstOrCreateAccount(user string, password string, module ModuleInterface) *Account
	GetAccount(module ModuleInterface) *Account

	// OAuth2 client storage functionality

	CreateOAuthClient(id string, secret string, accessToken string, refreshToken string, module ModuleInterface)
	GetFirstOrCreateOAuthClient(
		id string, secret string, accessToken string, refreshToken string, module ModuleInterface,
	) *OAuthClient
	GetOAuthClient(module ModuleInterface) *OAuthClient

	// cookie storage functionality

	GetAllCookies(module ModuleInterface) (cookies []*Cookie)
	GetCookie(name string, module ModuleInterface) *Cookie
	GetFirstOrCreateCookie(name string, value string, expirationString string, module ModuleInterface) *Cookie
	CreateCookie(name string, value string, expiration sql.NullTime, module ModuleInterface)
	UpdateCookie(name string, value string, expirationString string, module ModuleInterface)
	UpdateCookieDisabledStatus(name string, disabled bool, module ModuleInterface)
}
