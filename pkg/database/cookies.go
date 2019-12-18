package database

import (
	"database/sql"

	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
)

func (db *DbIO) createCookiesTable(connection *sql.DB) (err error) {
	sqlStatement := `
		CREATE TABLE cookies
		(
			uid 		INTEGER PRIMARY KEY AUTOINCREMENT,
			name 		VARCHAR(255) 	DEFAULT '',
			value 		VARCHAR(255) 	DEFAULT '',
			expiration 	TIMESTAMP 		DEFAULT NULL,
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
		rows *sql.Rows
		err  error
	)

	if module != nil {
		stmt, err := db.connection.Prepare(`
			SELECT * FROM cookies
			WHERE NOT disabled
			  AND (CURRENT_TIMESTAMP < datetime(expiration, 'unixepoch') OR expiration IS NULL)
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
			  AND (CURRENT_TIMESTAMP < datetime(expiration, 'unixepoch') OR expiration IS NULL)
			ORDER BY module, uid`)
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
		  AND (CURRENT_TIMESTAMP < datetime(expiration, 'unixepoch') OR expiration IS NULL)
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
