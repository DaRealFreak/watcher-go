package models

import "database/sql"

// DatabaseInterface of used functions from the application to eventually change the underlying library
type DatabaseInterface interface {
	// tracked item storage functionality
	GetTrackedItems(module ModuleInterface, includeCompleted bool) []*TrackedItem
	GetFirstOrCreateTrackedItem(uri string, module ModuleInterface) *TrackedItem
	CreateTrackedItem(uri string, module ModuleInterface)
	ChangeTrackedItemCompleteStatus(trackedItem *TrackedItem, complete bool)

	// account storage functionality
	CreateAccount(user string, password string, module ModuleInterface)
	GetFirstOrCreateAccount(user string, password string, module ModuleInterface) *Account
	GetAccount(module ModuleInterface) *Account
	UpdateTrackedItem(trackedItem *TrackedItem, currentItem string)

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

// Account contains all required data from accounts in the application
type Account struct {
	ID       int
	Module   string
	Username string
	Password string
	Disabled bool
}

// TrackedItem contains all required data from tracked items in the application
type TrackedItem struct {
	ID          int
	URI         string
	CurrentItem string
	Module      string
	Complete    bool
}
