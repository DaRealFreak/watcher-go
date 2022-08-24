package watcher

import "github.com/DaRealFreak/watcher-go/internal/raven"

// AddAccountByURI extracts the module based on the uri and adds an account if not registered already
func (app *Watcher) AddAccountByURI(uri string, user string, password string) {
	module := app.ModuleFactory.GetModuleFromURI(uri)
	app.DbCon.GetFirstOrCreateAccount(user, password, module)
}

// AddItemByURI adds an item based on the uri and sets it to the passed current item if not nil
func (app *Watcher) AddItemByURI(uri string, currentItem string) {
	module := app.ModuleFactory.GetModuleFromURI(uri)
	normalizedUri, err := module.ModuleInterface.AddItem(uri)
	raven.CheckError(err)

	trackedItems := app.DbCon.GetAllOrCreateTrackedItemIgnoreSubFolder(normalizedUri, module)

	for _, trackedItem := range trackedItems {
		if len(trackedItems) == 0 {
			app.DbCon.UpdateTrackedItem(trackedItem, currentItem)
		}
	}
}

// AddOAuthClientByURI adds an OAuth2 client based on the uri
func (app *Watcher) AddOAuthClientByURI(
	uri string, clientID string, clientSecret string, accessToken string, refreshToken string,
) {
	module := app.ModuleFactory.GetModuleFromURI(uri)
	app.DbCon.GetFirstOrCreateOAuthClient(clientID, clientSecret, accessToken, refreshToken, module)
}

// AddCookieByURI adds a cookie based on the uri
func (app *Watcher) AddCookieByURI(uri string, name string, value string, expiration string) {
	module := app.ModuleFactory.GetModuleFromURI(uri)
	app.DbCon.GetFirstOrCreateCookie(name, value, expiration, module)
}
