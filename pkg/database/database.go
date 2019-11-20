// Package database is the implementation of the DatabaseInterface using SQLite3
package database

import (
	"database/sql"
	"os"

	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
	"github.com/spf13/viper"

	// import for side effects
	_ "github.com/mattn/go-sqlite3"
)

// DbIO implements the DatabaseInterface and contains the connection to the database
type DbIO struct {
	models.DatabaseInterface
	connection *sql.DB
}

// NewConnection initializes the database DatabaseConnection to our sqlite file.
// Creates the database if the looked up file doesn't exist yet
func NewConnection() *DbIO {
	dbIO := DbIO{}

	if _, err := os.Stat(viper.GetString("Database.Path")); os.IsNotExist(err) {
		dbIO.createDatabase()
	}

	db, err := sql.Open("sqlite3", viper.GetString("Database.Path")+"?_journal=WAL")
	raven.CheckError(err)

	dbIO.connection = db

	return &dbIO
}

// CloseConnection safely closes the database connection
func (db *DbIO) CloseConnection() {
	err := db.connection.Close()
	raven.CheckError(err)
}

// RemoveDatabase removes the currently set database, used primarily for unit tests
func RemoveDatabase() {
	if _, err := os.Stat(viper.GetString("Database.Path")); err == nil {
		err := os.Remove(viper.GetString("Database.Path"))
		if err != nil {
			panic(err)
		}
	}
}

// createDatabase creates the sqlite file and creates the required tables
func (db *DbIO) createDatabase() {
	connection, err := sql.Open("sqlite3", viper.GetString("Database.Path")+"?_journal=WAL")
	raven.CheckError(err)

	defer raven.CheckClosure(connection)

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
	raven.CheckError(err)

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
	raven.CheckError(err)

	sqlStatement = `
		CREATE TABLE oauth_clients
		(
			uid           INTEGER PRIMARY KEY AUTOINCREMENT,
			client_id     VARCHAR(255) DEFAULT '',
			client_secret VARCHAR(255) DEFAULT '',
			access_token  VARCHAR(255) DEFAULT '',
			refresh_token VARCHAR(255) DEFAULT '',
			auth_url      VARCHAR(255) DEFAULT '',
			token_url     VARCHAR(255) DEFAULT '',
			module        VARCHAR(255) DEFAULT '' NOT NULL ,
			disabled      BOOLEAN      DEFAULT FALSE NOT NULL
		);
	`

	_, err = connection.Exec(sqlStatement)
	raven.CheckError(err)
}
