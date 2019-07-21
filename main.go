package main

import (
	"log"
	"watcher/database"
)

// main functionality of the database
// logs fatal error if no DatabaseConnection could be established
func main() {
	dbCon := database.NewConnection()
	if dbCon.Connection.Error != nil {
		log.Fatal(dbCon.Connection.Error)
	}
	log.Printf("%q", &dbCon.Connection.Database)
	log.Printf("%q", &dbCon.Connection.Error)
}
