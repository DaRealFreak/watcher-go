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
			expiration 	DATETIME 		DEFAULT CURRENT_TIMESTAMP,
			module 		VARCHAR(255) 	DEFAULT '' NOT NULL,
			disabled 	BOOLEAN 		DEFAULT FALSE NOT NULL
		);
	`
	_, err = connection.Exec(sqlStatement)

	return err
}

// GetCookies retrieves all cookies associated to the passed module which is not expired or disabled
func (db *DbIO) GetCookies(module models.ModuleInterface) (cookies []*models.Cookie) {
	// ignore cookies matching on the second since it'll be expired already until we actually use it
	stmt, err := db.connection.Prepare(`
		SELECT * FROM cookies
		WHERE NOT disabled
		  AND CURRENT_TIMESTAMP < datetime(expiration, 'unixepoch')
		  AND module = ?
		ORDER BY uid
	`)
	raven.CheckError(err)

	rows, err := stmt.Query(module.ModuleKey())
	raven.CheckError(err)

	defer raven.CheckClosure(rows)

	for rows.Next() {
		var cookie models.Cookie

		raven.CheckError(rows.Scan(
			&cookie.ID, &cookie.Name, &cookie.Value, &cookie.Expiration, &cookie.Module, &cookie.Disabled,
		))

		cookies = append(cookies, &cookie)
	}

	return cookies
}
