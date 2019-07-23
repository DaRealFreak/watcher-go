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
	fmt.Println("all items regardless of module: ", dbCon.GetTrackedItems(nil))
	fmt.Println("all items of module: ", dbCon.GetTrackedItems(&module))
	exampleItem := dbCon.GetFirstOrCreateTrackedItem("test_item", "test_module")
	fmt.Println("example item persisted: ", exampleItem)
	dbCon.ChangeTrackedItemCompleteStatus(&exampleItem, true)
	fmt.Println("update example item completed", exampleItem)
	dbCon.ChangeTrackedItemCompleteStatus(&exampleItem, false)
	fmt.Println("update example item uncompleted", exampleItem)

	dbCon.CloseConnection()
}
