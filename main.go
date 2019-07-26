package main

import (
	"watcher-go/database"
	"watcher-go/http"
)

type watcher struct {
	dbCon   *database.DbIO
	session *http.Session
}

// main functionality of the database
// logs fatal error if no DatabaseConnection could be established
func main() {
	watcher := watcher{
		dbCon:   database.NewConnection(),
		session: http.NewSession(),
	}

	// ToDO: iterate through all active items and run them
	watcher.dbCon.CloseConnection()
}

// extract the module based on the uri and add account if not registered already
func (app watcher) AddAccountByUri(uri string, user string, password string) {
	// ToDo: implement functionality to retrieve the module based on the uri
	module := uri
	app.dbCon.GetFirstOrCreateAccount(user, password, module)
}

// add item based on the uri and set it to the passed current item if not nil
func (app watcher) AddItemByUri(uri string, currentItem string) {
	// ToDo: implement functionality to retrieve the module based on the uri
	module := uri
	trackedItem := app.dbCon.GetFirstOrCreateTrackedItem(uri, module)
	if currentItem != "" {
		app.dbCon.UpdateTrackedItem(&trackedItem, currentItem)
	}
}
