package main

import (
	"fmt"
	"watcher-go/database"
)

// main functionality of the database
// logs fatal error if no DatabaseConnection could be established
func main() {
	dbCon := database.NewConnection()
	module := "test"
	fmt.Println(dbCon.GetItems(nil))
	fmt.Println(dbCon.GetItems(&module))
	dbCon.GetFirstOrCreateItem("test_item", "test_module")
}
