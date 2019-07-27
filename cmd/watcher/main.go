package main

import (
	"watcher-go/cmd/watcher/database"
	"watcher-go/cmd/watcher/modules"
)

type watcher struct {
	dbCon         *database.DbIO
	moduleFactory *modules.ModuleFactory
}

// main functionality of the database
// logs fatal error if no DatabaseConnection could be established
func main() {
	watcher := watcher{
		dbCon:         database.NewConnection(),
		moduleFactory: modules.NewModuleFactory(),
	}

	watcher.AddAccountByUri("https://chan.sankakucomplex.com/", "user", "name")
	module, _ := watcher.moduleFactory.GetModuleFromUri("https://chan.sankakucomplex.com/")
	account := watcher.dbCon.GetAccount(module)
	module.Module.Login(account)

	// ToDO: iterate through all active items and run them
	watcher.dbCon.CloseConnection()
}

// extract the module based on the uri and add account if not registered already
func (app watcher) AddAccountByUri(uri string, user string, password string) {
	module, err := app.moduleFactory.GetModuleFromUri(uri)
	app.checkError(err)

	app.dbCon.GetFirstOrCreateAccount(user, password, module)
}

// add item based on the uri and set it to the passed current item if not nil
func (app watcher) AddItemByUri(uri string, currentItem string) {
	module, err := app.moduleFactory.GetModuleFromUri(uri)
	app.checkError(err)

	trackedItem := app.dbCon.GetFirstOrCreateTrackedItem(uri, module)
	if currentItem != "" {
		app.dbCon.UpdateTrackedItem(trackedItem, currentItem)
	}
}

func (app watcher) checkError(err error) {
	if err != nil {
		panic(err)
	}
}
