package database

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
)

type dbIO struct {
	connection *sql.DB
}

// initializes the database DatabaseConnection to our sqlite file
// creates the database if the looked up file doesn't exist yet
func NewConnection() *dbIO {
	dbIO := dbIO{}
	if _, err := os.Stat("./watcher.db"); os.IsNotExist(err) {
		dbIO.createDatabase()
	}
	db, err := sql.Open("sqlite3", "./watcher.db")
	checkErr(err)

	dbIO.connection = db
	return &dbIO
}

// creates the sqlite file and creates the needed tables
func (dbIO) createDatabase() {
	db, err := sql.Open("sqlite3", "./watcher.db")
	checkErr(err)
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
	checkErr(err)

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
	checkErr(err)
}

// retrieve all tracked items from the sqlite database
// if module is set limit the results use the passed module as restraint
func (db dbIO) GetItems(module *string) []map[string]interface{} {
	var items []map[string]interface{}

	var rows *sql.Rows
	var err error
	if module == nil {
		rows, err = db.connection.Query("SELECT * FROM tracked_items WHERE NOT complete ORDER BY module, uid")
		checkErr(err)
	} else {
		stmt, err := db.connection.Prepare("SELECT * FROM tracked_items WHERE NOT complete AND module = ? ORDER BY uid")
		checkErr(err)

		rows, err = stmt.Query(*module)
		checkErr(err)
	}

	var uid int
	var uri string
	var currentItem string
	var usedModule string
	var complete bool
	for rows.Next() {
		err = rows.Scan(&uid, &uri, &currentItem, &usedModule, &complete)
		checkErr(err)

		items = append(items, map[string]interface{}{
			"uid":          uid,
			"uri":          uri,
			"current_item": currentItem,
			"module":       usedModule,
			"complete":     complete,
		})
	}

	err = rows.Close()
	checkErr(err)

	return items
}

// extracted function to check for an error, log fatal always on database errors
func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
