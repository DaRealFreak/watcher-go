package database

import (
	"testing"

	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/stretchr/testify/assert"
)

func TestDbIO_GetCookies(t *testing.T) {
	_, err := dbIO.connection.Exec(`
		INSERT INTO cookies (uid, name, value, expiration, module, disabled) 
		VALUES (1, 'PHPSESSID', '123456_test123456', '1608256840000', 'test.module', 0);`,
	)
	assert.New(t).NoError(err)

	_, err = dbIO.connection.Exec(`
		INSERT INTO cookies (uid, name, value, expiration, module, disabled) 
		VALUES (2, 'device_token', 'test123456', '1608256840000', 'test.module', 0);`,
	)
	assert.New(t).NoError(err)

	cookies := dbIO.GetCookies(&models.Module{Key: "test.module"})
	assert.New(t).Equal(len(cookies), 2)
}
