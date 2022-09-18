// Package models contains structs and default functions used all over the application to avoid circular dependencies
package models

import (
	"fmt"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/configuration"
	internalHttp "github.com/DaRealFreak/watcher-go/internal/http"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
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
	// Load initializes the module, logs in and updates the progress of the process
	Load() error
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
	Initialized    bool
	LoggedIn       bool
	TriedLogin     bool
	URISchemas     []*regexp.Regexp
	ProxyLoopIndex int
	Cfg            *configuration.AppConfiguration
}

type ModuleNotImplementedError struct {
	error
}

func (e ModuleNotImplementedError) Error() string {
	return "module is not implemented yet"
}

// AddItem gives the module the option to parse the uri before adding it to the database (f.e. for normalizing)
func (t *Module) AddItem(uri string) (string, error) {
	parsedUrl, err := url.Parse(uri)
	if err != nil {
		return uri, err
	}

	return parsedUrl.String(), nil
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

	if len(sessionCookies) > 0 && t.Session != nil && t.Session.GetClient() != nil {
		t.Session.GetClient().Jar.SetCookies(sessionUrl, sessionCookies)
	}
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
				fp.TruncateMaxLength(fp.SanitizePath(trackedItem.SubFolder, false)),
				fp.TruncateMaxLength(fp.SanitizePath(data.DownloadTag, false)),
				fp.TruncateMaxLength(fp.SanitizePath(data.FileName, false)),
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

// GetDownloadDirectory returns the module download directory if set, else the default directory is getting returned
func (t *Module) GetDownloadDirectory() string {
	moduleDirectory := viper.GetString(fmt.Sprintf("Modules.%s.download.directory", t.GetViperModuleKey()))
	if moduleDirectory != "" {
		return moduleDirectory
	}

	return viper.GetString("download.directory")
}

// GetViperModuleKey returns the module key without "." characters since they'll ruin the generated tree structure
func (t *Module) GetViperModuleKey() string {
	return strings.ReplaceAll(t.Key, ".", "_")
}

func (t *Module) Load() error {
	if t.Initialized {
		return nil
	}

	t.InitializeModule()

	if t.TriedLogin || t.LoggedIn {
		return nil
	}

	account := t.DbIO.GetAccount(t)

	// no account available but module requires a login
	if account == nil {
		if t.RequiresLogin {
			log.WithField("module", t.Key).Errorf(
				"module requires a login, but no account could be found",
			)
		} else {
			t.Initialized = true
			return nil
		}
	}

	log.WithField("module", t.Key).Info(
		fmt.Sprintf("logging in for module %s", t.Key),
	)

	// login into the module
	if t.Login(account) {
		log.WithField("module", t.Key).Info("login successful")
	} else {
		if t.RequiresLogin {
			log.WithField("module", t.Key).Fatalf(
				"module requires a login, but the login failed",
			)
		} else {
			log.WithField("module", t.Key).Fatalf("login not successful")
		}
	}

	t.Initialized = true

	return nil
}
