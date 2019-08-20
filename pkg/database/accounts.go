package database

import (
	"database/sql"

	"github.com/DaRealFreak/watcher-go/pkg/models"

	// import for side effects
	_ "github.com/mattn/go-sqlite3"
)

// retrieve the first not disabled account of the passed module
func (db DbIO) GetAccount(module models.ModuleInterface) *models.Account {
	stmt, err := db.connection.Prepare("SELECT * FROM accounts WHERE NOT disabled AND module = ? ORDER BY uid")
	db.checkErr(err)

	rows, err := stmt.Query(module.Key())
	db.checkErr(err)
	defer rows.Close()

	if rows.Next() {
		account := models.Account{}
		err = rows.Scan(&account.ID, &account.Username, &account.Password, &account.Module, &account.Disabled)
		db.checkErr(err)
		return &account
	}
	return nil
}

// retrieve all accounts of only by module if module is not nil
func (db DbIO) GetAllAccounts(module models.ModuleInterface) (accounts []*models.Account) {
	var rows *sql.Rows
	var err error
	if module != nil {
		stmt, err := db.connection.Prepare("SELECT * FROM accounts WHERE NOT disabled AND module = ? ORDER BY module, uid")
		db.checkErr(err)

		rows, err = stmt.Query(module.Key())
		db.checkErr(err)
	} else {
		rows, err = db.connection.Query("SELECT * FROM accounts WHERE NOT disabled ORDER BY module, uid")
	}
	db.checkErr(err)

	for rows.Next() {
		account := models.Account{}
		err := rows.Scan(&account.ID, &account.Username, &account.Password, &account.Module, &account.Disabled)
		db.checkErr(err)

		accounts = append(accounts, &account)
	}
	return accounts
}

// check if an account exists already, if not create it
// returns the already persisted or the newly created account
func (db DbIO) GetFirstOrCreateAccount(user string, password string, module models.ModuleInterface) *models.Account {
	stmt, err := db.connection.Prepare("SELECT * FROM accounts WHERE user = ? AND module = ?")
	db.checkErr(err)

	rows, err := stmt.Query(user, module.Key())
	db.checkErr(err)
	defer rows.Close()

	if rows.Next() {
		account := models.Account{}
		// item already persisted
		err = rows.Scan(&account.ID, &account.Username, &account.Password, &account.Module, &account.Disabled)
		db.checkErr(err)
		return &account
	}
	// create the item and call the same function again
	db.CreateAccount(user, password, module)
	return db.GetFirstOrCreateAccount(user, password, module)
}

// inserts the passed user and password of the specific module into the database
func (db DbIO) CreateAccount(user string, password string, module models.ModuleInterface) {
	stmt, err := db.connection.Prepare("INSERT INTO accounts (user, password, module) VALUES (?, ?, ?)")
	db.checkErr(err)
	defer stmt.Close()

	_, err = stmt.Exec(user, password, module.Key())
	db.checkErr(err)
}

// updates the password of the passed user/module entry
func (db DbIO) UpdateAccount(user string, password string, module models.ModuleInterface) {
	stmt, err := db.connection.Prepare("UPDATE accounts SET password = ? WHERE user = ? AND module = ?")
	db.checkErr(err)
	defer stmt.Close()

	_, err = stmt.Exec(password, user, module.Key())
	db.checkErr(err)
}

// disables the account of the passed user/module
func (db DbIO) UpdateAccountDisabledStatus(user string, disabled bool, module models.ModuleInterface) {
	var disabledInt int8
	if disabled {
		disabledInt = 1
	} else {
		disabledInt = 0
	}
	stmt, err := db.connection.Prepare("UPDATE accounts SET disabled = ? WHERE user = ? AND module = ?")
	db.checkErr(err)
	defer stmt.Close()

	_, err = stmt.Exec(disabledInt, user, module.Key())
	db.checkErr(err)
}
