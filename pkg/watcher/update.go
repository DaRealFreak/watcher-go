package watcher

// UpdateAccountDisabledStatusByURI updates an account of the passed uri and changes the disabled status
func (app *Watcher) UpdateAccountDisabledStatusByURI(uri string, user string, disabled bool) {
	module := app.ModuleFactory.GetModuleFromURI(uri)
	app.DbCon.UpdateAccountDisabledStatus(user, disabled, module)
}

// UpdateAccountByURI updates the password of an account of the passed uri
func (app *Watcher) UpdateAccountByURI(uri string, user string, password string) {
	module := app.ModuleFactory.GetModuleFromURI(uri)
	app.DbCon.UpdateAccount(user, password, module)
}

// UpdateOAuthClientDisabledStatusByURI updates an OAuth2 client of the passed uri and changes the disabled status
func (app *Watcher) UpdateOAuthClientDisabledStatusByURI(uri string, clientID string, token string, disabled bool) {
	module := app.ModuleFactory.GetModuleFromURI(uri)
	app.DbCon.UpdateOAuthClientDisabledStatus(clientID, token, disabled, module)
}

// UpdateOAuthClientByURI updates the OAuth2 client of the passed uri
func (app *Watcher) UpdateOAuthClientByURI(
	uri string, clientID string, clientSecret string, accessToken string, refreshToken string,
) {
	module := app.ModuleFactory.GetModuleFromURI(uri)
	app.DbCon.UpdateOAuthClient(clientID, clientSecret, accessToken, refreshToken, module)
}
