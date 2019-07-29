package main

import (
	"fmt"
	"github.com/kubernetes/klog"
	"watcher-go/cmd/watcher/arguments"
	"watcher-go/cmd/watcher/database"
	"watcher-go/cmd/watcher/modules"
)

type watcher struct {
	dbCon         *database.DbIO
	moduleFactory *modules.ModuleFactory
}

func init() {
	klog.InitFlags(nil)
}

// main functionality, iterates through all tracked items and parses them
func main() {
	dbIO := database.NewConnection()
	watcher := watcher{
		dbCon:         dbIO,
		moduleFactory: modules.NewModuleFactory(dbIO),
	}

	if *arguments.Account != "" && *arguments.Password != "" && *arguments.Uri != "" {
		watcher.AddAccountByUri(*arguments.Uri, *arguments.Account, *arguments.Password)
		return
	} else if *arguments.Uri != "" && (*arguments.Account == "" || *arguments.Password == "") {
		watcher.AddItemByUri(*arguments.Uri, *arguments.CurrentItem)
	} else {
		for _, item := range watcher.dbCon.GetTrackedItems(nil) {
			module := watcher.moduleFactory.GetModule(item.Module)
			if !module.IsLoggedIn() {
				klog.Info(fmt.Sprintf("logging in for module %s", module.Key()))
				account := watcher.dbCon.GetAccount(module)
				success := module.Login(account)
				if success {
					klog.Info("login successful")
				} else {
					klog.Warning("login not successful")
				}
			}
			klog.Info(fmt.Sprintf("parsing item %s (current id: %s)", item.Uri, item.CurrentItem))
			module.Parse(item)
		}
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
