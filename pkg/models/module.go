package models

import (
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/DaRealFreak/watcher-go/pkg/raven"

	"github.com/DaRealFreak/watcher-go/pkg/http"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/x-cray/logrus-prefixed-formatter"
)

// ModuleInterface of used functions from the application for all modules
type ModuleInterface interface {
	// Key returns the module key
	Key() (key string)
	// RequiresLogin checks if this module requires a login to work
	RequiresLogin() (requiresLogin bool)
	// IsLoggedIn returns the logged in status
	IsLoggedIn() bool
	// RegisterURISchema adds our pattern to the URI Schemas
	RegisterURISchema(uriSchemas map[string][]*regexp.Regexp)
	// Login logs us in for the current session if possible/account available
	Login(account *Account) (success bool)
	// Parse parses the tracked item
	Parse(item *TrackedItem)
}

// DownloadQueueItem is a generic struct in case the module doesn't require special actions
type DownloadQueueItem struct {
	ItemID      string
	DownloadTag string
	FileName    string
	FileURI     string
}

// Module is an implementation to the ModuleInterface to provide basic functions/variables
type Module struct {
	ModuleInterface
	DbIO     DatabaseInterface
	Session  http.SessionInterface
	LoggedIn bool
	Logger   *log.Logger
}

// GetFileName retrieves the file name of a passed uri
func (t *Module) GetFileName(uri string) string {
	parsedURI, _ := url.Parse(uri)
	return filepath.Base(parsedURI.Path)
}

// GetFileExtension retrieves the file extension of a passed uri
func (t *Module) GetFileExtension(uri string) string {
	parsedURI, _ := url.Parse(uri)
	return filepath.Ext(parsedURI.Path)
}

// ReverseDownloadQueueItems reverses the download queue items to get the oldest items first
// to be able to interrupt the update process anytime
func (t *Module) ReverseDownloadQueueItems(downloadQueue []DownloadQueueItem) []DownloadQueueItem {
	for i, j := 0, len(downloadQueue)-1; i < j; i, j = i+1, j-1 {
		downloadQueue[i], downloadQueue[j] = downloadQueue[j], downloadQueue[i]
	}
	return downloadQueue
}

// ProcessDownloadQueue processes the default download queue, can be used if the module doesn't require special actions
func (t *Module) ProcessDownloadQueue(downloadQueue []DownloadQueueItem, trackedItem *TrackedItem) {
	log.Info(fmt.Sprintf("found %d new items for uri: \"%s\"", len(downloadQueue), trackedItem.URI))

	for index, data := range downloadQueue {
		log.Info(
			fmt.Sprintf(
				"downloading updates for uri: \"%s\" (%0.2f%%)",
				trackedItem.URI,
				float64(index+1)/float64(len(downloadQueue))*100,
			),
		)
		err := t.Session.DownloadFile(
			path.Join(viper.GetString("download.directory"), t.Key(), data.DownloadTag, data.FileName),
			data.FileURI,
		)
		raven.CheckError(err)
		t.DbIO.UpdateTrackedItem(trackedItem, data.ItemID)
	}
}

// SanitizePath replaces reserved characters https://en.wikipedia.org/wiki/Filename#Reserved_characters_and_words
// and trims the result
func (t *Module) SanitizePath(path string, allowSeparator bool) string {
	var reservedCharacters *regexp.Regexp
	if allowSeparator {
		reservedCharacters = regexp.MustCompile("[:\"*?<>|]+")
	} else {
		reservedCharacters = regexp.MustCompile("[\\\\/:\"*?<>|]+")
	}
	path = reservedCharacters.ReplaceAllString(path, "_")
	for strings.Contains(path, "__") {
		path = strings.Replace(path, "__", "_", -1)
	}
	path = strings.Trim(path, "_")
	return path
}

// moduleFormatter is a custom formatter for modules for a better
// overview during parallel module updates
type moduleFormatter struct {
	prefixed.TextFormatter
	moduleKey string
}

// Format prints the default Formatter with a module prefix
func (f *moduleFormatter) Format(entry *log.Entry) ([]byte, error) {
	result, err := f.TextFormatter.Format(entry)
	return append([]byte("["+f.moduleKey+"]"), result...), err
}

// SetFormattedLogger sets the formatted logger for the module and its session
func (t *Module) SetFormattedLogger() {
	standardLogger := log.StandardLogger()
	t.Logger = &log.Logger{
		Out:   standardLogger.Out,
		Hooks: standardLogger.Hooks,
		Formatter: &moduleFormatter{
			TextFormatter: prefixed.TextFormatter{
				TimestampFormat: "2006-01-02 15:04:05",
				FullTimestamp:   true,
				ForceColors:     true,
				ForceFormatting: true,
			},
			moduleKey: t.Key(),
		},
		ReportCaller: standardLogger.ReportCaller,
		Level:        standardLogger.Level,
		ExitFunc:     standardLogger.ExitFunc,
	}
	t.Session.SetLogger(t.Logger)
}
