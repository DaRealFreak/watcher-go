package database

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/stretchr/testify/assert"
)

func seedCookiesTable(t *testing.T) {
	futureTimestamp := time.Now().AddDate(1, 0, 0)
	pastTimestamp := time.Now().AddDate(-1, 0, 0)
	currentTimestamp := time.Now()

	// truncate table
	_, err := dbIO.connection.Exec(`DELETE FROM cookies WHERE uid;`)
	assert.New(t).NoError(err)

	// nolint: gosec
	_, err = dbIO.connection.Exec(fmt.Sprintf(`
		INSERT INTO cookies (name, expiration, module)
		VALUES ('future_cookie', '%s', 'test.module');`,
		futureTimestamp.Format(time.RFC3339),
	))
	assert.New(t).NoError(err)

	// nolint: gosec
	_, err = dbIO.connection.Exec(fmt.Sprintf(`
		INSERT INTO cookies (name, expiration, module)
		VALUES ('current_cookie', '%s', 'test.module');`,
		currentTimestamp.Format(time.RFC3339),
	))
	assert.New(t).NoError(err)

	// nolint: gosec
	_, err = dbIO.connection.Exec(fmt.Sprintf(`
		INSERT INTO cookies (name, expiration, module)
		VALUES ('expired_cookie', '%s', 'test.module');`,
		pastTimestamp.Format(time.RFC3339),
	))
	assert.New(t).NoError(err)

	_, err = dbIO.connection.Exec(`
		INSERT INTO cookies (name, expiration, module)
		VALUES ('null_cookie', null, 'test.module');`,
	)
	assert.New(t).NoError(err)

	// nolint: gosec
	_, err = dbIO.connection.Exec(fmt.Sprintf(`
		INSERT INTO cookies (name, expiration, module)
		VALUES ('future_cookie_2', '%s', 'test.module.2');`,
		futureTimestamp.Format(time.RFC3339),
	))
	assert.New(t).NoError(err)

	// nolint: gosec
	_, err = dbIO.connection.Exec(fmt.Sprintf(`
		INSERT INTO cookies (name, expiration, module)
		VALUES ('expired_cookie_2', '%s', 'test.module.2');`,
		pastTimestamp.Format(time.RFC3339),
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

func TestDbIO_CreateCookie(t *testing.T) {
	// truncate table
	_, err := dbIO.connection.Exec(`DELETE FROM cookies WHERE uid;`)
	assert.New(t).NoError(err)

	cookies := dbIO.GetAllCookies(nil)
	assert.New(t).Equal(0, len(cookies))

	// create normal cookie
	dbIO.CreateCookie("name", "value", sql.NullTime{
		Time:  time.Now().AddDate(1, 0, 0),
		Valid: true,
	}, &models.Module{Key: "test.module"})

	cookies = dbIO.GetAllCookies(&models.Module{Key: "test.module"})
	assert.New(t).Equal(1, len(cookies))

	// create null expiration cookie
	dbIO.CreateCookie("name", "value", sql.NullTime{}, &models.Module{Key: "test.module"})

	cookies = dbIO.GetAllCookies(&models.Module{Key: "test.module"})
	assert.New(t).Equal(2, len(cookies))
}

func TestDbIO_GetFirstOrCreateCookie(t *testing.T) {
	// truncate table
	_, err := dbIO.connection.Exec(`DELETE FROM cookies WHERE uid;`)
	assert.New(t).NoError(err)

	cookies := dbIO.GetAllCookies(nil)
	assert.New(t).Equal(0, len(cookies))

	dbIO.GetFirstOrCreateCookie(
		"test", "name", time.Now().Add(24*time.Hour).Format(time.RFC1123), &models.Module{Key: "test.module"},
	)

	cookies = dbIO.GetAllCookies(&models.Module{Key: "test.module"})
	assert.New(t).Equal(1, len(cookies))
}

func TestDbIO_getNullTimeFromString(t *testing.T) {
	validTestStrings := []string{
		// FireFox copy result on cookie expiration attribute
		`Expires:"Sat, 19 Dec 2020 17:16:43 GMT"`,
		// Chrome copy result on cookie expiration attribute
		"2054-12-31T23:59:59.284Z",
		// Edge/IE has no direct copy functionality (lol), but displays it like this
		"Wed, 18 Dec 2019 13:57:18 GMT",
	}

	for _, testString := range validTestStrings {
		assert.New(t).Equal(true, dbIO.getNullTimeFromString(testString).Valid)
	}
}
