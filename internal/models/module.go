// Package models contains structs and default functions used all over the application to avoid circular dependencies
package models

import (
	"fmt"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/configuration"
	internalHttp "github.com/DaRealFreak/watcher-go/internal/http"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// ModuleInterface of used functions from the application for all modules
type ModuleInterface interface {
	// ModuleKey retrieve the module key
	ModuleKey() string
	// InitializeModule initializes the registered bare module
	InitializeModule()
	// AddModuleCommand option for the modules to register custom settings/commands
	AddModuleCommand(command *cobra.Command)
	// Login logs us in for the current session if possible/account available
	Login(account *Account) (success bool)
	// Parse parses the tracked item
	Parse(item *TrackedItem) error
	// AddItem gives the module the option to parse the uri before adding it to the database (f.e. for normalizing)
	AddItem(uri string) (string, error)
}

// DownloadQueueItem is a generic struct in case the module doesn't require special actions
type DownloadQueueItem struct {
	ItemID          string
	DownloadTag     string
	FileName        string
	FileURI         string
	FallbackFileURI string
}

// Module is an implementation to the ModuleInterface to provide basic functions/variables
type Module struct {
	ModuleInterface
	DbIO           DatabaseInterface
	Session        internalHttp.SessionInterface
	Key            string
	RequiresLogin  bool
	LoggedIn       bool
	TriedLogin     bool
	URISchemas     []*regexp.Regexp
	ProxyLoopIndex int
	Cfg            *configuration.AppConfiguration
}

// AddItem gives the module the option to parse the uri before adding it to the database (f.e. for normalizing)
func (t *Module) AddItem(uri string) (string, error) {
	return uri, nil
}

// ModuleKey returns the key of the module required to use as interface to prevent import cycles
func (t *Module) ModuleKey() string {
	return t.Key
}

// RegisterURISchema registers the URI schemas of the module to the passed map
func (t *Module) RegisterURISchema(uriSchemas map[string][]*regexp.Regexp) {
	uriSchemas[t.Key] = t.URISchemas
}

// SetDbIO sets the database IO implementation
func (t *Module) SetDbIO(io DatabaseInterface) {
	t.DbIO = io
}

// SetCfg sets the app configuration for each module
func (t *Module) SetCfg(cfg *configuration.AppConfiguration) {
	t.Cfg = cfg
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

func (t *Module) SetCookies() {
	sessionUrl, err := url.Parse(fmt.Sprintf("https://%s", t.Key))
	if err != nil {
		return
	}

	cookies := t.DbIO.GetAllCookies(t)
	sessionCookies := make([]*http.Cookie, len(cookies))
	for i, cookie := range cookies {
		sessionCookies[i] = &http.Cookie{
			Name:   cookie.Name,
			Value:  cookie.Value,
			Domain: sessionUrl.Host,
		}
	}

	t.Session.GetClient().Jar.SetCookies(sessionUrl, sessionCookies)
}

// ProcessDownloadQueue processes the default download queue, can be used if the module doesn't require special actions
func (t *Module) ProcessDownloadQueue(downloadQueue []DownloadQueueItem, trackedItem *TrackedItem) error {
	log.WithField("module", t.Key).Info(
		fmt.Sprintf("found %d new items for uri: \"%s\"", len(downloadQueue), trackedItem.URI),
	)

	for index, data := range downloadQueue {
		log.WithField("module", t.Key).Info(
			fmt.Sprintf(
				"downloading updates for uri: \"%s\" (%0.2f%%)",
				trackedItem.URI,
				float64(index+1)/float64(len(downloadQueue))*100,
			),
		)

		err := t.Session.DownloadFile(
			path.Join(
				viper.GetString("download.directory"),
				t.Key,
				t.TruncateMaxLength(t.SanitizePath(data.DownloadTag, false)),
				t.TruncateMaxLength(t.SanitizePath(data.FileName, false)),
			),
			data.FileURI,
		)
		if err != nil {
			return err
		}

		t.DbIO.UpdateTrackedItem(trackedItem, data.ItemID)
	}

	return nil
}

// SanitizePath replaces reserved characters https://en.wikipedia.org/wiki/Filename#Reserved_characters_and_words
// and trims the result
func (t Module) SanitizePath(path string, allowSeparator bool) string {
	var reservedCharacters *regexp.Regexp
	if allowSeparator {
		reservedCharacters = regexp.MustCompile("[:\"*?<>|]+")
	} else {
		reservedCharacters = regexp.MustCompile("[\\\\/:\"*?<>|]+")
	}

	path = reservedCharacters.ReplaceAllString(path, "_")
	path = strings.ReplaceAll(path, "\t", " ")
	for strings.Contains(path, "__") {
		path = strings.Replace(path, "__", "_", -1)
	}

	for strings.Contains(path, "..") {
		path = strings.Replace(path, "..", ".", -1)
	}

	path = strings.Trim(path, "_")

	return strings.Trim(path, " ")
}

// GetViperModuleKey returns the module key without "." characters since they'll ruin the generated tree structure
func (t *Module) GetViperModuleKey() string {
	return strings.ReplaceAll(t.Key, ".", "_")
}
