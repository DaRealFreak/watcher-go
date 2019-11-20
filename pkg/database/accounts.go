package database

import (
	"database/sql"

	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/DaRealFreak/watcher-go/pkg/raven"

	// import for side effects
	_ "github.com/mattn/go-sqlite3"
)

// GetAccount retrieves the first not disabled account of the passed module
func (db *DbIO) GetAccount(module models.ModuleInterface) *models.Account {
	stmt, err := db.connection.Prepare("SELECT * FROM accounts WHERE NOT disabled AND module = ? ORDER BY uid")
	raven.CheckError(err)

	rows, err := stmt.Query(module.ModuleKey())
	raven.CheckError(err)

	defer raven.CheckClosure(rows)

	if rows.Next() {
		account := models.Account{}
		err = rows.Scan(&account.ID, &account.Username, &account.Password, &account.Module, &account.Disabled)
		raven.CheckError(err)

		return &account
	}

	return nil
}

// GetAllAccounts retrieves all accounts of only by module if module is not nil
func (db *DbIO) GetAllAccounts(module models.ModuleInterface) (accounts []*models.Account) {
	var (
		rows *sql.Rows
		err  error
	)

	if module != nil {
		stmt, err := db.connection.Prepare("SELECT * FROM accounts WHERE NOT disabled AND module = ? ORDER BY module, uid")
		raven.CheckError(err)

		rows, err = stmt.Query(module.ModuleKey())
		raven.CheckError(err)
	} else {
		rows, err = db.connection.Query("SELECT * FROM accounts WHERE NOT disabled ORDER BY module, uid")
	}

	raven.CheckError(err)

	for rows.Next() {
		account := models.Account{}
		err := rows.Scan(&account.ID, &account.Username, &account.Password, &account.Module, &account.Disabled)
		raven.CheckError(err)

		accounts = append(accounts, &account)
	}

	return accounts
}

// GetFirstOrCreateAccount checks if an account exists already, else creates it
// returns the already persisted or the newly created account
func (db *DbIO) GetFirstOrCreateAccount(user string, password string, module models.ModuleInterface) *models.Account {
	stmt, err := db.connection.Prepare("SELECT * FROM accounts WHERE user = ? AND module = ?")
	raven.CheckError(err)

	rows, err := stmt.Query(user, module.ModuleKey())
	raven.CheckError(err)

	defer raven.CheckClosure(rows)

	if rows.Next() {
		account := models.Account{}
		// item already persisted
		err = rows.Scan(&account.ID, &account.Username, &account.Password, &account.Module, &account.Disabled)
		raven.CheckError(err)

		return &account
	}

	// create the item and call the same function again
	db.CreateAccount(user, password, module)

	return db.GetFirstOrCreateAccount(user, password, module)
}

// CreateAccount inserts the passed user and password of the specific module into the database
func (db *DbIO) CreateAccount(user string, password string, module models.ModuleInterface) {
	stmt, err := db.connection.Prepare("INSERT INTO accounts (user, password, module) VALUES (?, ?, ?)")
	raven.CheckError(err)

	defer raven.CheckClosure(stmt)

	_, err = stmt.Exec(user, password, module.ModuleKey())
	raven.CheckError(err)
}

// UpdateAccount updates the password of the passed user/module entry
func (db *DbIO) UpdateAccount(user string, password string, module models.ModuleInterface) {
	stmt, err := db.connection.Prepare("UPDATE accounts SET password = ? WHERE user = ? AND module = ?")
	raven.CheckError(err)

	defer raven.CheckClosure(stmt)

	_, err = stmt.Exec(password, user, module.ModuleKey())
	raven.CheckError(err)
}

// UpdateAccountDisabledStatus disables the account of the passed user/module
func (db *DbIO) UpdateAccountDisabledStatus(user string, disabled bool, module models.ModuleInterface) {
	var disabledInt int8

	if disabled {
		disabledInt = 1
	} else {
		disabledInt = 0
	}

	stmt, err := db.connection.Prepare("UPDATE accounts SET disabled = ? WHERE user = ? AND module = ?")
	raven.CheckError(err)

	defer raven.CheckClosure(stmt)

	_, err = stmt.Exec(disabledInt, user, module.ModuleKey())
	raven.CheckError(err)
}
