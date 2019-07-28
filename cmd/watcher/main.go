package main

import (
	"fmt"
	"github.com/golang/glog"
	"watcher-go/cmd/watcher/database"
	"watcher-go/cmd/watcher/modules"
)

type watcher struct {
	dbCon         *database.DbIO
	moduleFactory *modules.ModuleFactory
}

// main functionality, iterates through all tracked items and parses them
func main() {
	databaseConnection := database.NewConnection()
	watcher := watcher{
		dbCon:         databaseConnection,
		moduleFactory: modules.NewModuleFactory(databaseConnection),
	}

	for _, item := range watcher.dbCon.GetTrackedItems(nil) {
		module := watcher.moduleFactory.GetModule(item.Module)
		if !module.IsLoggedIn() {
			glog.Info(fmt.Sprintf("logging in for module %s", module.Key()))
			account := watcher.dbCon.GetAccount(module)
			success := module.Login(account)
			if success {
				glog.Info("login successful")
			} else {
				glog.Warning("login not successful")
			}
		}
		glog.Info(fmt.Sprintf("parsing item %s (current id: %s)", item.Uri, item.CurrentItem))
		module.Parse(item)
	}

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
