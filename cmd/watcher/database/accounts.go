package database

import (
	_ "github.com/mattn/go-sqlite3"
	"watcher-go/cmd/watcher/models"
)

// retrieve the first not disabled account of the passed module
func (db DbIO) GetAccount(module *models.Module) *models.Account {
	stmt, err := db.connection.Prepare("SELECT * FROM accounts WHERE NOT disabled AND module = ? ORDER BY uid")
	db.checkErr(err)

	rows, err := stmt.Query(module.Key())
	db.checkErr(err)
	defer rows.Close()

	if rows.Next() {
		account := models.Account{}
		err = rows.Scan(&account.Id, &account.Username, &account.Password, &account.Module, &account.Disabled)
		db.checkErr(err)
		return &account
	} else {
		return nil
	}
}

// check if an account exists already, if not create it
// returns the already persisted or the newly created account
func (db DbIO) GetFirstOrCreateAccount(user string, password string, module *models.Module) *models.Account {
	stmt, err := db.connection.Prepare("SELECT * FROM accounts WHERE user = ? AND module = ?")
	db.checkErr(err)

	rows, err := stmt.Query(user, module.Key())
	db.checkErr(err)
	defer rows.Close()

	if rows.Next() {
		account := models.Account{}
		// item already persisted
		err = rows.Scan(&account.Id, &account.Username, &account.Password, &account.Module, &account.Disabled)
		db.checkErr(err)
		return &account
	} else {
		// create the item and call the same function again
		db.CreateAccount(user, password, module)
		return db.GetFirstOrCreateAccount(user, password, module)
	}
}

// inserts the passed user and password of the specific module into the database
func (db DbIO) CreateAccount(user string, password string, module *models.Module) {
	stmt, err := db.connection.Prepare("INSERT INTO accounts (user, password, module) VALUES (?, ?, ?)")
	db.checkErr(err)
	defer stmt.Close()

	_, err = stmt.Exec(user, password, module.Key())
	db.checkErr(err)
}
