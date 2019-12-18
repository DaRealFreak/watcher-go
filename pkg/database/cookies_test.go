package database

import (
	"fmt"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestDbIO_GetCookies(t *testing.T) {
	futureTimestamp := time.Now().AddDate(1, 0, 0).Unix()
	pastTimestamp := time.Now().AddDate(-1, 0, 0).Unix()
	currentTimestamp := time.Now().Unix()

	_, err := dbIO.connection.Exec(fmt.Sprintf(`
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

	cookies := dbIO.GetCookies(&models.Module{Key: "test.module"})
	assert.New(t).Equal(1, len(cookies))
}
