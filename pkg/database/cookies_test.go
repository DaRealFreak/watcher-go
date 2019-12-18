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
		INSERT INTO cookies (name, expiration, module)
		VALUES ('future_cookie', '%d', 'test.module');`,
		futureTimestamp,
	))
	assert.New(t).NoError(err)

	_, err = dbIO.connection.Exec(fmt.Sprintf(`
		INSERT INTO cookies (name, expiration, module)
		VALUES ('current_cookie', '%d', 'test.module');`,
		currentTimestamp,
	))
	assert.New(t).NoError(err)

	_, err = dbIO.connection.Exec(fmt.Sprintf(`
		INSERT INTO cookies (name, expiration, module)
		VALUES ('expired_cookie', '%d', 'test.module');`,
		pastTimestamp,
	))
	assert.New(t).NoError(err)

	_, err = dbIO.connection.Exec(`
		INSERT INTO cookies (name, expiration, module)
		VALUES ('null_cookie', null, 'test.module');`,
	)
	assert.New(t).NoError(err)

	_, err = dbIO.connection.Exec(fmt.Sprintf(`
		INSERT INTO cookies (name, expiration, module)
		VALUES ('future_cookie_2', '%d', 'test.module.2');`,
		futureTimestamp,
	))
	assert.New(t).NoError(err)

	_, err = dbIO.connection.Exec(fmt.Sprintf(`
		INSERT INTO cookies (name, expiration, module)
		VALUES ('expired_cookie_2', '%d', 'test.module.2');`,
		pastTimestamp,
	))
	assert.New(t).NoError(err)

	_, err = dbIO.connection.Exec(`
		INSERT INTO cookies (name, expiration, module)
		VALUES ('null_cookie', null, 'test.module.2');`,
	)
	assert.New(t).NoError(err)

}

func TestDbIO_GetCookie(t *testing.T) {
	seedCookiesTable(t)

	cookie := dbIO.GetCookie("future_cookie", &models.Module{Key: "test.module"})
	assert.New(t).NotEmpty(cookie)

	cookie = dbIO.GetCookie("current_cookie", &models.Module{Key: "test.module"})
	assert.New(t).Empty(cookie)

	cookie = dbIO.GetCookie("expired_cookie", &models.Module{Key: "test.module"})
	assert.New(t).Empty(cookie)

	cookie = dbIO.GetCookie("null_cookie", &models.Module{Key: "test.module"})
	assert.New(t).NotEmpty(cookie)
}

func TestDbIO_GetAllCookies(t *testing.T) {
	seedCookiesTable(t)

	cookies := dbIO.GetAllCookies(&models.Module{Key: "test.module"})
	assert.New(t).Equal(2, len(cookies))

	cookies = dbIO.GetAllCookies(nil)
	assert.New(t).Equal(4, len(cookies))
}
