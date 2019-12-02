// Package models contains structs and default functions used all over the application to avoid circular dependencies
package models

import (
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/DaRealFreak/watcher-go/pkg/http"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// ModuleInterface of used functions from the application for all modules
type ModuleInterface interface {
	// retrieve the module key
	ModuleKey() string
	// initializes the registered bare module
	InitializeModule()
	// option for the modules to register custom settings/commands
	AddSettingsCommand(command *cobra.Command)
	// Login logs us in for the current session if possible/account available
	Login(account *Account) (success bool)
	// Parse parses the tracked item
	Parse(item *TrackedItem) error
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
	DbIO           DatabaseInterface
	Session        http.SessionInterface
	Key            string
	RequiresLogin  bool
	LoggedIn       bool
	TriedLogin     bool
	URISchemas     []*regexp.Regexp
	ProxyLoopIndex int
}

// ModuleKey returns the key of the module required to use as interface to prevent import cycles
func (t *Module) ModuleKey() string {
	return t.Key
}

// RegisterURISchema registers the URI schemas of the module to the passed map
func (t *Module) RegisterURISchema(URISchemas map[string][]*regexp.Regexp) {
	URISchemas[t.Key] = t.URISchemas
}

// SetDbIO sets the database IO implementation
func (t *Module) SetDbIO(io DatabaseInterface) {
	t.DbIO = io
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
			path.Join(viper.GetString("download.directory"), t.Key, data.DownloadTag, data.FileName),
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

	for strings.Contains(path, "..") {
		path = strings.Replace(path, "..", ".", -1)
	}

	path = strings.Trim(path, "_")

	return path
}

// GetViperModuleKey returns the module key without "." characters since they'll ruin the generated tree structure
func (t *Module) GetViperModuleKey() string {
	return strings.ReplaceAll(t.Key, ".", "_")
}
