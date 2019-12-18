package database

import (
	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
)

// GetCookies retrieves all cookies associated to the passed module which is not expired or disabled
func (db *DbIO) GetCookies(module models.ModuleInterface) (cookies []*models.Cookie) {
	stmt, err := db.connection.Prepare("SELECT * FROM cookies WHERE NOT disabled AND module = ? ORDER BY uid")
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
