package raven

import (
	"database/sql"
	"io"
	"time"

	"github.com/spf13/viper"

	"github.com/getsentry/sentry-go"
	log "github.com/sirupsen/logrus"
)

// SetupSentry initializes the sentry
func SetupSentry() {
	sentryDsn := ""
	// only set DSN if user opted in for that
	if viper.GetBool("watcher.sentry") {
		sentryDsn = "https://3ad96038aedf4d859c95f8ae755617ec@sentry.io/1535770"
	}

	if err := sentry.Init(sentry.ClientOptions{
		Dsn: sentryDsn,
	}); err != nil {
		log.Fatal(err)
	}
}

// CheckError checks if the passed error is not nil and passes it to the sentry DSN
func CheckError(err error) {
	if err != nil {
		sentry.CaptureException(err)
		// Since sentry emits events in the background we need to make sure
		// they are sent before we shut down
		sentry.Flush(time.Second * 5)
		log.Fatal(err)
	}
}

// CheckDbClosure checks for errors on deferred DB connections
func CheckDbClosure(db *sql.DB) {
	CheckError(db.Close())
}

// CheckRowClosure checks for errors on deferred Rows
func CheckRowClosure(row *sql.Rows) {
	CheckError(row.Close())
}

// CheckRowClosure checks for errors on deferred Statements
func CheckStatementClosure(stmt *sql.Stmt) {
	CheckError(stmt.Close())
}

// CheckRowClosure checks for errors on deferred ReadCloser objects
func CheckReadCloser(closer io.ReadCloser) {
	CheckError(closer.Close())
}
