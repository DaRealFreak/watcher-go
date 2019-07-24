package main

import (
	"watcher-go/database"
)

type watcher struct {
	dbCon *database.DbIO
}

// main functionality of the database
// logs fatal error if no DatabaseConnection could be established
func main() {
	watcher := watcher{}
	watcher.dbCon = database.NewConnection()
	// ToDO: iterate through all active items and run them
	watcher.dbCon.CloseConnection()
}

// extract the module based on the uri and add account if not registered already
func (w watcher) AddAccountByUri(uri string, user string, password string) {
	// ToDo: implement functionality to retrieve the module based on the uri
	module := uri
	w.dbCon.GetFirstOrCreateAccount(user, password, module)
}

// add item based on the uri and set it to the passed current item if not nil
func (w watcher) AddItemByUri(uri string, currentItem string) {
	// ToDo: implement functionality to retrieve the module based on the uri
	module := uri
	trackedItem := w.dbCon.GetFirstOrCreateTrackedItem(uri, module)
	if currentItem != "" {
		w.dbCon.UpdateTrackedItem(&trackedItem, currentItem)
	}
}
