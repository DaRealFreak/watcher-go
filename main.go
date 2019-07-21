package main

import (
	"watcher/database"
)

// main functionality of the database
// logs fatal error if no DatabaseConnection could be established
func main() {
	dbCon := database.NewConnection()
	module := "test"
	dbCon.GetItems(nil)
	dbCon.GetItems(&module)
}
