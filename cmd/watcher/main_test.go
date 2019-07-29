package main

import (
	"fmt"
	"os"
	"testing"
	"watcher-go/cmd/watcher/database"
	"watcher-go/cmd/watcher/modules"
)

var app watcher

// main function for tests
// remove previous database to prevent previous data influencing the tests
// and remove the database at the end again for a clean system
func TestMain(m *testing.M) {
	// constructor
	database.RemoveDatabase()
	dbIO := database.NewConnection()
	app = watcher{
		dbCon:         dbIO,
		moduleFactory: modules.NewModuleFactory(dbIO),
	}

	// run the unit tests
	code := m.Run()

	// destructor
	app.dbCon.CloseConnection()
	database.RemoveDatabase()
	os.Exit(code)
}

// test the add account function
func TestAddAccountByUri(t *testing.T) {
	testUri := "https://chan.sankakucomplex.com/"
	app.AddAccountByUri(testUri, "user", "pass")

	module, _ := app.moduleFactory.GetModuleFromUri(testUri)
	account := app.dbCon.GetFirstOrCreateAccount("user", "different_pass", module)
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
	module, _ := app.moduleFactory.GetModuleFromUri(testUri)
	fmt.Println("all items regardless of module: ", app.dbCon.GetTrackedItems(nil))
	fmt.Println("all items of module: ", app.dbCon.GetTrackedItems(module))

	exampleItem := app.dbCon.GetFirstOrCreateTrackedItem("test_item", module)
	fmt.Println("example item persisted: ", exampleItem)
	app.dbCon.ChangeTrackedItemCompleteStatus(exampleItem, true)
	fmt.Println("update example item completed", exampleItem)
	app.dbCon.ChangeTrackedItemCompleteStatus(exampleItem, false)
	fmt.Println("update example item uncompleted", exampleItem)
}
