package raven

import (
	"github.com/getsentry/sentry-go"
	log "github.com/sirupsen/logrus"
	"time"
)

// SetupSentry initializes the sentry
func SetupSentry() {
	if err := sentry.Init(sentry.ClientOptions{
		Dsn: "https://3ad96038aedf4d859c95f8ae755617ec@sentry.io/1535770",
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
