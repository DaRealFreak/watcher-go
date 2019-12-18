package models

// DatabaseInterface of used functions from the application to eventually change the underlying library
type DatabaseInterface interface {
	// tracked item functions
	GetTrackedItems(module ModuleInterface, includeCompleted bool) []*TrackedItem
	GetFirstOrCreateTrackedItem(uri string, module ModuleInterface) *TrackedItem
	CreateTrackedItem(uri string, module ModuleInterface)
	ChangeTrackedItemCompleteStatus(trackedItem *TrackedItem, complete bool)

	// account functions
	CreateAccount(user string, password string, module ModuleInterface)
	GetFirstOrCreateAccount(user string, password string, module ModuleInterface) *Account
	GetAccount(module ModuleInterface) *Account
	UpdateTrackedItem(trackedItem *TrackedItem, currentItem string)

	// OAuth client functions
	CreateOAuthClient(
		clientID string, clientSecret string, accessToken string, refreshToken string, module ModuleInterface,
	)
	GetFirstOrCreateOAuthClient(
		clientID string, clientSecret string, accessToken string, refreshToken string, module ModuleInterface,
	) *OAuthClient
	GetOAuthClient(module ModuleInterface) *OAuthClient

	// Cookie functions
	GetAllCookies(module ModuleInterface) (cookies []*Cookie)
	GetCookie(name string, module ModuleInterface) *Cookie
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
