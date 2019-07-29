package database

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"watcher-go/cmd/watcher/models"
)

type DbIO struct {
	models.DatabaseInterface
	connection *sql.DB
}

// initializes the database DatabaseConnection to our sqlite file
// creates the database if the looked up file doesn't exist yet
func NewConnection() *DbIO {
	dbIO := DbIO{}
	if _, err := os.Stat("./watcher.db"); os.IsNotExist(err) {
		dbIO.createDatabase()
	}
	db, err := sql.Open("sqlite3", "./watcher.db")
	dbIO.checkErr(err)

	dbIO.connection = db
	return &dbIO
}

// remove the whole database file
func RemoveDatabase() {
	if _, err := os.Stat("./watcher.db"); err == nil {
		err := os.Remove("./watcher.db")
		if err != nil {
			panic(err)
		}
	}
}

// close the connection
func (db DbIO) CloseConnection() {
	err := db.connection.Close()
	db.checkErr(err)
}

// creates the sqlite file and creates the needed tables
func (db DbIO) createDatabase() {
	connection, err := sql.Open("sqlite3", "./watcher.db")
	db.checkErr(err)
	defer connection.Close()

	sqlStatement := `
		CREATE TABLE accounts
		(
			uid      INTEGER      PRIMARY KEY AUTOINCREMENT,
			user     VARCHAR(255) DEFAULT '',
			password VARCHAR(255) DEFAULT '',
			module   VARCHAR(255) NOT NULL,
			disabled BOOLEAN      DEFAULT FALSE NOT NULL
		);
	`
	_, err = connection.Exec(sqlStatement)
	db.checkErr(err)

	sqlStatement = `
		CREATE TABLE tracked_items
		(
			uid          INTEGER PRIMARY KEY AUTOINCREMENT,
			uri          VARCHAR(255) DEFAULT '',
			current_item VARCHAR(255) DEFAULT '',
			module       VARCHAR(255) DEFAULT '' NOT NULL ,
			complete     BOOLEAN      DEFAULT FALSE NOT NULL
		);
	`
	_, err = connection.Exec(sqlStatement)
	db.checkErr(err)
}

// extracted function to check for an error, log fatal always on database errors
func (db DbIO) checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
