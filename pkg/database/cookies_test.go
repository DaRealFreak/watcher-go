package database

import (
	"fmt"
	"testing"
	"time"

	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/stretchr/testify/assert"
)

func seedCookiesTable(t *testing.T) {
	futureTimestamp := time.Now().AddDate(1, 0, 0).Unix()
	pastTimestamp := time.Now().AddDate(-1, 0, 0).Unix()
	currentTimestamp := time.Now().Unix()

	// truncate table
	_, err := dbIO.connection.Exec(`DELETE FROM cookies WHERE uid;`)
	assert.New(t).NoError(err)

	_, err = dbIO.connection.Exec(fmt.Sprintf(`
		INSERT INTO cookies (uid, name, value, expiration, module, disabled)
		VALUES (1, 'future_cookie', 'test123456', '%d', 'test.module', 0);`,
		futureTimestamp,
	))
	assert.New(t).NoError(err)

	_, err = dbIO.connection.Exec(fmt.Sprintf(`
		INSERT INTO cookies (uid, name, value, expiration, module, disabled)
		VALUES (2, 'current_cookie', 'test123456', '%d', 'test.module', 0);`,
		currentTimestamp,
	))
	assert.New(t).NoError(err)

	_, err = dbIO.connection.Exec(fmt.Sprintf(`
		INSERT INTO cookies (uid, name, value, expiration, module, disabled)
		VALUES (3, 'expired_cookie', 'test123456', '%d', 'test.module', 0);`,
		pastTimestamp,
	))
	assert.New(t).NoError(err)
}

func TestDbIO_GetCookies(t *testing.T) {
	seedCookiesTable(t)

	cookies := dbIO.GetCookies(&models.Module{Key: "test.module"})
	assert.New(t).Equal(1, len(cookies))
}

func TestDbIO_GetCookie(t *testing.T) {
	seedCookiesTable(t)

	cookie := dbIO.GetCookie("future_cookie", &models.Module{Key: "test.module"})
	assert.New(t).NotNil(cookie)

	cookie = dbIO.GetCookie("current_cookie", &models.Module{Key: "test.module"})
	assert.New(t).Empty(cookie)

	cookie = dbIO.GetCookie("expired_cookie", &models.Module{Key: "test.module"})
	assert.New(t).Empty(cookie)
}
