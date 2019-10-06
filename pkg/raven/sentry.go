package raven

import (
	"io"
	"time"

	"github.com/DaRealFreak/watcher-go/pkg/version"
	"github.com/getsentry/sentry-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// SetupSentry initializes the sentry
func SetupSentry() {
	sentryDsn := ""
	// only set DSN if user opted in for that
	if viper.GetBool("watcher.sentry") {
		sentryDsn = "https://3ad96038aedf4d859c95f8ae755617ec@sentry.io/1535770"
	}

	if err := sentry.Init(sentry.ClientOptions{
		Dsn:     sentryDsn,
		Release: "watcher-go@" + version.VERSION,
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

// CheckErrorNonFatal checks if the passed error is not nil and pass it to the sentry DSN, contrary to CheckError
// the log will not be fatal resulting in the application to continue running
func CheckErrorNonFatal(err error) {
	if err != nil {
		sentry.CaptureException(err)
		// Since sentry emits events in the background we need to make sure
		// they are sent before we warn the user and continue
		sentry.Flush(time.Second * 5)
		log.Warning(err)
	}
}

// CheckClosure checks for errors on closeable objects
func CheckClosure(obj io.Closer) {
	CheckError(obj.Close())
}

// CheckClosureNonFatal checks for errors on closeable objects simply warning the user and not exiting the application
func CheckClosureNonFatal(obj io.Closer) {
	CheckErrorNonFatal(obj.Close())
}
