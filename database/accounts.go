package database

import (
	_ "github.com/mattn/go-sqlite3"
)

type Account struct {
	Id       int
	Module   string
	Username string
	Password string
	Disabled bool
}

// retrieve the first not disabled account of the passed module
func (db DbIO) GetAccount(module string) *Account {
	stmt, err := db.connection.Prepare("SELECT * FROM accounts WHERE NOT disabled AND module = ? ORDER BY uid")
	db.checkErr(err)

	rows, err := stmt.Query(module)
	db.checkErr(err)
	defer rows.Close()

	if rows.Next() {
		account := Account{}
		err = rows.Scan(&account.Id, &account.Username, &account.Password, &account.Module, &account.Disabled)
		db.checkErr(err)
		return &account
	} else {
		return nil
	}
}

// check if an account exists already, if not create it
// returns the already persisted or the newly created account
func (db DbIO) GetFirstOrCreateAccount(user string, password string, module string) Account {
	stmt, err := db.connection.Prepare("SELECT * FROM accounts WHERE user = ? AND module = ?")
	db.checkErr(err)

	rows, err := stmt.Query(user, module)
	db.checkErr(err)
	defer rows.Close()

	if rows.Next() {
		account := Account{}
		// item already persisted
		err = rows.Scan(&account.Id, &account.Username, &account.Password, &account.Module, &account.Disabled)
		db.checkErr(err)
		return account
	} else {
		// create the item and call the same function again
		db.CreateAccount(user, password, module)
		return db.GetFirstOrCreateAccount(user, password, module)
	}
}

// inserts the passed user and password of the specific module into the database
func (db DbIO) CreateAccount(user string, password string, module string) {
	stmt, err := db.connection.Prepare("INSERT INTO accounts (user, password, module) VALUES (?, ?, ?)")
	db.checkErr(err)
	defer stmt.Close()

	_, err = stmt.Exec(user, password, module)
	db.checkErr(err)
}
