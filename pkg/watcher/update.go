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
