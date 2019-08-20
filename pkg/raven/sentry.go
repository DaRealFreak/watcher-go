package raven

import (
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
