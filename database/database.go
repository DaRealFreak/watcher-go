package database

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
)

type dbIO struct {
	Connection connection
}

type connection struct {
	Database *sql.DB
	Error    error
}

// initializes the database DatabaseConnection to our sqlite file
// creates the database if the looked up file doesn't exist yet
func NewConnection() *dbIO {
	dbIO := dbIO{}
	if _, err := os.Stat("./watcher.db"); os.IsNotExist(err) {
		dbIO.createDatabase()
	}
	db, err := sql.Open("sqlite3", "./watcher.db")
	dbIO.Connection = connection{db, err}
	return &dbIO
}

// creates the sqlite file and creates the needed tables
func (dbIO) createDatabase() {
	db, err := sql.Open("sqlite3", "./watcher.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	sqlStatement := `
		CREATE TABLE accounts
		(
			module   VARCHAR(255) NOT NULL PRIMARY KEY,
			user     VARCHAR(255) DEFAULT '',
			password VARCHAR(255) DEFAULT ''
		);
	`
	_, err = db.Exec(sqlStatement)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStatement)
		return
	}

	sqlStatement = `
		CREATE TABLE tracked_items
		(
			uid          INTEGER PRIMARY KEY AUTOINCREMENT,
			uri          VARCHAR(255) DEFAULT '',
			current_item VARCHAR(255) DEFAULT '',
			module       VARCHAR(255) DEFAULT '' NOT NULL ,
			complete     BOOLEAN      default FALSE NOT NULL 
		);
	`
	_, err = db.Exec(sqlStatement)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStatement)
		return
	}
}
