package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/raven"
)

func (db *DbIO) createCookiesTable(connection *sql.DB) (err error) {
	sqlStatement := `
		CREATE TABLE cookies
		(
			uid 		INTEGER PRIMARY KEY AUTOINCREMENT,
			name 		VARCHAR(255) 	DEFAULT '',
			value 		VARCHAR(255) 	DEFAULT '',
			expiration 	DATETIME 		DEFAULT NULL,
			module 		VARCHAR(255) 	DEFAULT '' NOT NULL,
			disabled 	BOOLEAN 		DEFAULT FALSE NOT NULL
		);
	`
	_, err = connection.Exec(sqlStatement)

	return err
}

// GetAllCookies retrieves all cookies of only by module if module is not nil
func (db *DbIO) GetAllCookies(module models.ModuleInterface) (cookies []*models.Cookie) {
	var (
		stmt *sql.Stmt
		rows *sql.Rows
		err  error
	)

	if module != nil {
		stmt, err = db.connection.Prepare(`
			SELECT * FROM cookies
			WHERE NOT disabled
			  AND (strftime('%s','now') < expiration OR expiration = 0)
			  AND module = ?
			ORDER BY uid
		`)
		raven.CheckError(err)

		rows, err = stmt.Query(module.ModuleKey())
		raven.CheckError(err)
	} else {
		rows, err = db.connection.Query(`
			SELECT * 
			FROM cookies 
			WHERE NOT disabled
			  AND (strftime('%s','now') < expiration OR expiration = 0)
			ORDER BY module, uid
		`)
	}

	raven.CheckError(err)

	for rows.Next() {
		var cookie models.Cookie

		raven.CheckError(rows.Scan(
			&cookie.ID, &cookie.Name, &cookie.Value, &cookie.Expiration, &cookie.Module, &cookie.Disabled,
		))

		cookies = append(cookies, &cookie)
	}

	return cookies
}

// GetCookie retrieves a specific cookie associated to the passed module which is not expired or disabled
func (db *DbIO) GetCookie(name string, module models.ModuleInterface) *models.Cookie {
	stmt, err := db.connection.Prepare(`
		SELECT * FROM cookies
		WHERE NOT disabled
		  AND (strftime('%s','now') < expiration OR expiration = 0)
		  AND name = ?
		  AND module = ?
		ORDER BY uid
	`)
	raven.CheckError(err)

	rows, err := stmt.Query(name, module.ModuleKey())
	raven.CheckError(err)

	defer raven.CheckClosure(rows)

	if rows.Next() {
		var cookie models.Cookie

		raven.CheckError(rows.Scan(
			&cookie.ID, &cookie.Name, &cookie.Value, &cookie.Expiration, &cookie.Module, &cookie.Disabled,
		))

		return &cookie
	}

	return nil
}

// GetFirstOrCreateCookie checks if a cookie exists already, else creates it
// returns the already persisted or the newly created cookie
func (db *DbIO) GetFirstOrCreateCookie(
	name string, value string, expirationString string, module models.ModuleInterface,
) *models.Cookie {
	expiration := db.getNullTimeFromString(expirationString)

	stmt, err := db.connection.Prepare("SELECT * FROM cookies WHERE name = ? AND module = ?")
	raven.CheckError(err)

	rows, err := stmt.Query(name, module.ModuleKey())
	raven.CheckError(err)

	defer raven.CheckClosure(rows)

	if rows.Next() {
		var cookie models.Cookie

		raven.CheckError(rows.Scan(
			&cookie.ID, &cookie.Name, &cookie.Value, &cookie.Expiration, &cookie.Module, &cookie.Disabled,
		))

		return &cookie
	}

	// create the item and call the same function again
	db.CreateCookie(name, value, expiration, module)

	return db.GetFirstOrCreateCookie(name, value, expirationString, module)
}

// CreateCookie inserts the passed name, value and expiration of the specific module into the cookies table
func (db *DbIO) CreateCookie(name string, value string, expiration sql.NullTime, module models.ModuleInterface) {
	stmt, err := db.connection.Prepare("INSERT INTO cookies (name, value, expiration, module) VALUES (?, ?, ?, ?)")
	raven.CheckError(err)

	defer raven.CheckClosure(stmt)

	_, err = stmt.Exec(name, value, db.getUnixTimestampFromNullTime(expiration), module.ModuleKey())
	raven.CheckError(err)
}

// UpdateCookie updates the value and/or the expiration date of the passed name/module associated cookie
func (db *DbIO) UpdateCookie(name string, value string, expirationString string, module models.ModuleInterface) {
	expiration := db.getNullTimeFromString(expirationString)

	stmt, err := db.connection.Prepare("UPDATE cookies SET value = ?, expiration = ? WHERE name = ? AND module = ?")
	raven.CheckError(err)

	defer raven.CheckClosure(stmt)

	_, err = stmt.Exec(value, db.getUnixTimestampFromNullTime(expiration), name, module.ModuleKey())
	raven.CheckError(err)
}

// UpdateCookieDisabledStatus disables or enables the cookie of the passed user/module
func (db *DbIO) UpdateCookieDisabledStatus(name string, disabled bool, module models.ModuleInterface) {
	var disabledInt int8

	if disabled {
		disabledInt = 1
	} else {
		disabledInt = 0
	}

	stmt, err := db.connection.Prepare("UPDATE cookies SET disabled = ? WHERE name = ? AND module = ?")
	raven.CheckError(err)

	defer raven.CheckClosure(stmt)

	_, err = stmt.Exec(disabledInt, name, module.ModuleKey())
	raven.CheckError(err)
}

func (db *DbIO) getNullTimeFromString(timeString string) sql.NullTime {
	supportedLayouts := []string{
		time.ANSIC,
		time.UnixDate,
		time.RubyDate,
		time.RFC822,
		time.RFC822Z,
		time.RFC850,
		time.RFC1123,
		time.RFC1123Z,
		time.RFC3339,
		fmt.Sprintf(`Expires:"%s"`, time.RFC1123),
	}

	for _, layouts := range supportedLayouts {
		parsedTime, err := time.Parse(layouts, timeString)
		if err == nil {
			return sql.NullTime{
				Time:  parsedTime,
				Valid: true,
			}
		}
	}

	// return empty NullTime (ending as null in the database)
	return sql.NullTime{}
}

func (db *DbIO) getUnixTimestampFromNullTime(time sql.NullTime) int64 {
	if time.Time.Unix() < 0 {
		return 0
	}

	return time.Time.Unix()
}
