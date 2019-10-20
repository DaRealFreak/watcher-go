package models

import (
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/DaRealFreak/watcher-go/pkg/http"
	"github.com/DaRealFreak/watcher-go/pkg/http/session"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	DbIO     DatabaseInterface
	Session  http.SessionInterface
	LoggedIn bool
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
	log.WithField("module", t.Key()).Info(
		fmt.Sprintf("found %d new items for uri: \"%s\"", len(downloadQueue), trackedItem.URI),
	)

	for index, data := range downloadQueue {
		log.WithField("module", t.Key()).Info(
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

// getViperModuleKey returns the module key without "." characters since they'll ruin the generated tree structure
func (t *Module) getViperModuleKey() string {
	return strings.ReplaceAll(t.Key(), ".", "_")
}

// addProxyCommands adds the module specific commands for the proxy server
func (t *Module) AddProxyCommands(command *cobra.Command) {
	var proxySettings session.ProxySettings

	proxyCmd := &cobra.Command{
		Use:   "proxy",
		Short: "proxy configurations",
		Long:  "options to configure proxy settings used for the module",
		Run: func(cmd *cobra.Command, args []string) {
			// enable proxy after changing the settings
			viper.Set(fmt.Sprintf("Modules.%s.Proxy.Enable", t.getViperModuleKey()), true)
			viper.Set(fmt.Sprintf("Modules.%s.Proxy.Host", t.getViperModuleKey()), proxySettings.Address)
			viper.Set(fmt.Sprintf("Modules.%s.Proxy.Port", t.getViperModuleKey()), proxySettings.Port)
			viper.Set(fmt.Sprintf("Modules.%s.Proxy.Username", t.getViperModuleKey()), proxySettings.Username)
			viper.Set(fmt.Sprintf("Modules.%s.Proxy.Password", t.getViperModuleKey()), proxySettings.Password)
			raven.CheckError(viper.WriteConfig())
		},
	}

	proxyCmd.Flags().StringVarP(
		&proxySettings.Address,
		"host", "H", "",
		"host of the proxy server (required)",
	)
	proxyCmd.Flags().IntVarP(
		&proxySettings.Port,
		"port", "P", 1080,
		"port of the proxy server",
	)
	proxyCmd.Flags().StringVarP(
		&proxySettings.Username,
		"user", "u", "",
		"username for the proxy server",
	)
	proxyCmd.Flags().StringVarP(
		&proxySettings.Password,
		"password", "p", "",
		"password for the proxy server",
	)

	_ = proxyCmd.MarkFlagRequired("host")

	// add sub commands for proxy command
	t.addEnableProxyCommand(proxyCmd)
	t.addDisableProxyCommand(proxyCmd)

	// add proxy command to parent command
	command.AddCommand(proxyCmd)
}

// addEnableProxyCommand adds the proxy enable sub command
func (t *Module) addEnableProxyCommand(command *cobra.Command) {
	enableCmd := &cobra.Command{
		Use:   "enable",
		Short: "enables proxy usage",
		Long:  "option to enable proxy server usage again, after it got manually disabled",
		Run: func(cmd *cobra.Command, args []string) {
			viper.Set("Modules.ehentai.Proxy.Enable", true)
			raven.CheckError(viper.WriteConfig())
		},
	}
	command.AddCommand(enableCmd)
}

// addDisableProxyCommand adds the proxy disable sub command
func (t *Module) addDisableProxyCommand(command *cobra.Command) {
	enableCmd := &cobra.Command{
		Use:   "disable",
		Short: "disables proxy usage",
		Long:  "option to disable proxy server usage",
		Run: func(cmd *cobra.Command, args []string) {
			viper.Set("Modules.ehentai.Proxy.Enable", false)
			raven.CheckError(viper.WriteConfig())
		},
	}
	command.AddCommand(enableCmd)
}

// getProxySettings returns the proxy settings for the module
func (t *Module) GetProxySettings() *session.ProxySettings {
	return &session.ProxySettings{
		Use:      viper.GetBool(fmt.Sprintf("Modules.%s.Proxy.Enable", t.getViperModuleKey())),
		Address:  viper.GetString(fmt.Sprintf("Modules.%s.Proxy.Host", t.getViperModuleKey())),
		Port:     viper.GetInt(fmt.Sprintf("Modules.%s.Proxy.Port", t.getViperModuleKey())),
		Username: viper.GetString(fmt.Sprintf("Modules.%s.Proxy.Username", t.getViperModuleKey())),
		Password: viper.GetString(fmt.Sprintf("Modules.%s.Proxy.Password", t.getViperModuleKey())),
	}
}
