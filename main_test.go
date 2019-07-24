package main

import (
	"fmt"
	"os"
	"testing"
	"watcher-go/database"
)

var w watcher

// main function for tests
// remove previous database to prevent previous data influencing the tests
// and remove the database at the end again for a clean system
func TestMain(m *testing.M) {
	// constructor
	database.RemoveDatabase()
	w = watcher{}
	w.dbCon = database.NewConnection()

	// run the unit tests
	code := m.Run()

	// destructor
	w.dbCon.CloseConnection()
	database.RemoveDatabase()
	os.Exit(code)
}

// test the add account function
func TestAddAccountByUri(t *testing.T) {
	w.AddAccountByUri("https://www.test.com", "user", "pass")

	account := w.dbCon.GetFirstOrCreateAccount("user", "different_pass", "https://www.test.com")
	if account.Password != "pass" {
		t.Fatal("password got updated or different user got added")
	}
}

// test the add item by uri function
func TestAddItemByUri(t *testing.T) {
	w.AddItemByUri("test_item", "")
	w.AddItemByUri("test_item_with_current_item", "hi there")

	// ToDo: check generated items
	module := "test_item"
	fmt.Println("all items regardless of module: ", w.dbCon.GetTrackedItems(nil))
	fmt.Println("all items of module: ", w.dbCon.GetTrackedItems(&module))

	exampleItem := w.dbCon.GetFirstOrCreateTrackedItem("test_item", module)
	fmt.Println("example item persisted: ", exampleItem)
	w.dbCon.ChangeTrackedItemCompleteStatus(&exampleItem, true)
	fmt.Println("update example item completed", exampleItem)
	w.dbCon.ChangeTrackedItemCompleteStatus(&exampleItem, false)
	fmt.Println("update example item uncompleted", exampleItem)
}
