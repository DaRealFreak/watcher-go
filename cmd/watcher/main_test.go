package main

import (
	"fmt"
	"os"
	"testing"
	"watcher-go/pkg/database"
	"watcher-go/pkg/watcher"
)

var app *watcher.Watcher

// main function for tests
// remove previous database to prevent previous data influencing the tests
// and remove the database at the end again for a clean system
func TestMain(m *testing.M) {
	// constructor
	database.RemoveDatabase()
	app = watcher.NewWatcher()

	// run the unit tests
	code := m.Run()

	// destructor
	app.DbCon.CloseConnection()
	database.RemoveDatabase()
	os.Exit(code)
}

// test the add account function
func TestAddAccountByUri(t *testing.T) {
	testUri := "https://chan.sankakucomplex.com/"
	app.AddAccountByUri(testUri, "user", "pass")

	module := app.ModuleFactory.GetModuleFromUri(testUri)
	account := app.DbCon.GetFirstOrCreateAccount("user", "different_pass", module)
	if account.Password != "pass" {
		t.Fatal("password got updated or different user got added")
	}
}

// test the add item by uri function
func TestAddItemByUri(t *testing.T) {
	testUri := "https://chan.sankakucomplex.com/"
	app.AddItemByUri(testUri, "")
	app.AddItemByUri(testUri+"_another", "hi there")

	// ToDo: check generated items
	module := app.ModuleFactory.GetModuleFromUri(testUri)
	fmt.Println("all items regardless of module: ", app.DbCon.GetTrackedItems(nil))
	fmt.Println("all items of module: ", app.DbCon.GetTrackedItems(module))

	exampleItem := app.DbCon.GetFirstOrCreateTrackedItem("test_item", module)
	fmt.Println("example item persisted: ", exampleItem)
	app.DbCon.ChangeTrackedItemCompleteStatus(exampleItem, true)
	fmt.Println("update example item completed", exampleItem)
	app.DbCon.ChangeTrackedItemCompleteStatus(exampleItem, false)
	fmt.Println("update example item uncompleted", exampleItem)
}
