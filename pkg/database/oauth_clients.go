package database

import (
	"database/sql"

	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/DaRealFreak/watcher-go/pkg/raven"

	// import for side effects
	_ "github.com/mattn/go-sqlite3"
)

// GetOAuthClient retrieves the first not disabled OAuth client of the passed module
func (db *DbIO) GetOAuthClient(module models.ModuleInterface) *models.OAuthClient {
	stmt, err := db.connection.Prepare(
		"SELECT * FROM oauth_clients WHERE NOT disabled AND module = ? ORDER BY uid",
	)
	raven.CheckError(err)

	rows, err := stmt.Query(module.ModuleKey())
	raven.CheckError(err)

	defer raven.CheckClosure(rows)

	if rows.Next() {
		oAuthClient := models.OAuthClient{}
		raven.CheckError(rows.Scan(
			&oAuthClient.ID, &oAuthClient.ClientID, &oAuthClient.ClientSecret,
			&oAuthClient.AccessToken, &oAuthClient.RefreshToken,
			&oAuthClient.Module, &oAuthClient.Disabled,
		))

		return &oAuthClient
	}

	return nil
}

// GetAllOAuthClients retrieves all OAuth clients of only by module if module is not nil
func (db *DbIO) GetAllOAuthClients(module models.ModuleInterface) (oAuthClients []*models.OAuthClient) {
	var (
		rows *sql.Rows
		err  error
	)

	if module != nil {
		stmt, err := db.connection.Prepare(
			"SELECT * FROM oauth_clients WHERE NOT disabled AND module = ? ORDER BY module, uid",
		)
		raven.CheckError(err)

		rows, err = stmt.Query(module.ModuleKey())
		raven.CheckError(err)
	} else {
		rows, err = db.connection.Query("SELECT * FROM oauth_clients WHERE NOT disabled ORDER BY module, uid")
	}

	raven.CheckError(err)

	for rows.Next() {
		oAuthClient := models.OAuthClient{}
		raven.CheckError(rows.Scan(
			&oAuthClient.ID, &oAuthClient.ClientID, &oAuthClient.ClientSecret,
			&oAuthClient.AccessToken, &oAuthClient.RefreshToken,
			&oAuthClient.Module, &oAuthClient.Disabled,
		))

		oAuthClients = append(oAuthClients, &oAuthClient)
	}

	return oAuthClients
}

// GetFirstOrCreateOAuthClient checks if an OAuth client exists already, else creates it
// returns the already persisted or the newly created OAuth client
func (db *DbIO) GetFirstOrCreateOAuthClient(
	clientID string, clientSecret string, accessToken string, refreshToken string, module models.ModuleInterface,
) *models.OAuthClient {
	stmt, err := db.connection.Prepare("SELECT * FROM oauth_clients WHERE client_id = ? AND module = ?")
	raven.CheckError(err)

	rows, err := stmt.Query(clientID, module.ModuleKey())
	raven.CheckError(err)

	defer raven.CheckClosure(rows)

	if rows.Next() {
		oAuthClient := models.OAuthClient{}
		// item already persisted
		raven.CheckError(rows.Scan(
			&oAuthClient.ID, &oAuthClient.ClientID, &oAuthClient.ClientSecret,
			&oAuthClient.AccessToken, &oAuthClient.RefreshToken,
			&oAuthClient.Module, &oAuthClient.Disabled,
		))

		return &oAuthClient
	}

	// create the item and call the same function again
	db.CreateOAuthClient(clientID, clientSecret, accessToken, refreshToken, module)

	return db.GetFirstOrCreateOAuthClient(clientID, clientSecret, accessToken, refreshToken, module)
}

// CreateOAuthClient inserts the passed OAuth client details of the specific module into the database
func (db *DbIO) CreateOAuthClient(
	clientID string, clientSecret string, accessToken string, refreshToken string, module models.ModuleInterface,
) {
	stmt, err := db.connection.Prepare(
		"INSERT INTO oauth_clients (client_id, client_secret, access_token, refresh_token, module) " +
			"VALUES (?, ?, ?, ?, ?)",
	)
	raven.CheckError(err)

	defer raven.CheckClosure(stmt)

	_, err = stmt.Exec(clientID, clientSecret, accessToken, refreshToken, module.ModuleKey())
	raven.CheckError(err)
}

// UpdateOAuthClientDisabledStatus disables the OAuth client of the passed client ID/module
func (db *DbIO) UpdateOAuthClientDisabledStatus(clientID string, disabled bool, module models.ModuleInterface) {
	var disabledInt int8

	if disabled {
		disabledInt = 1
	} else {
		disabledInt = 0
	}

	stmt, err := db.connection.Prepare(
		"UPDATE oauth_clients SET disabled = ? WHERE client_id = ? AND module = ?",
	)
	raven.CheckError(err)

	defer raven.CheckClosure(stmt)

	_, err = stmt.Exec(disabledInt, clientID, module.ModuleKey())
	raven.CheckError(err)
}
