// Package twitter contains the implementation of the twitter module
package twitter

import (
	"os"
	"regexp"
	"strings"

	formatter "github.com/DaRealFreak/colored-nested-formatter"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules"
	"github.com/DaRealFreak/watcher-go/internal/modules/twitter/api"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// twitter contains the implementation of the ModuleInterface
type twitter struct {
	*models.Module
	twitterAPI *api.TwitterAPI
}

// nolint: gochecknoinits
// init function registers the bare and the normal module to the module factories
func init() {
	modules.GetModuleFactory().RegisterModule(NewBareModule())
}

// NewBareModule returns a bare module implementation for the CLI options
func NewBareModule() *models.Module {
	module := &models.Module{
		Key:           "twitter.com",
		RequiresLogin: false,
		LoggedIn:      true,
		URISchemas: []*regexp.Regexp{
			regexp.MustCompile(".*twitter.com"),
		},
	}
	module.ModuleInterface = &twitter{
		Module: module,
	}

	// register module to log formatter
	formatter.AddFieldMatchColorScheme("module", &formatter.FieldMatch{
		Value: module.Key,
		Color: "232:39",
	})

	return module
}

// InitializeModule initializes the module
func (m *twitter) InitializeModule() {
	oauthClient := m.DbIO.GetOAuthClient(m)
	if oauthClient == nil || oauthClient.ClientID == "" || oauthClient.ClientSecret == "" {
		log.WithField("module", m.Key).Fatalf(
			"module requires an OAuth2 consumer ID and token",
		)
		// log.Fatal will already exit with error code 1, so the exit is just for the IDE here
		os.Exit(1)
	}

	m.twitterAPI = api.NewTwitterAPI(m.Key, oauthClient)
}

// AddModuleCommand adds custom module specific settings and commands to our application
func (m *twitter) AddModuleCommand(command *cobra.Command) {
	m.AddProxyCommands(command)
}

// Login logs us in for the current session if possible/account available
func (m *twitter) Login(_ *models.Account) bool {
	return m.LoggedIn
}

// Parse parses the tracked item
func (m *twitter) Parse(item *models.TrackedItem) error {
	return m.parsePage(item)
}

func (m *twitter) AddItem(uri string) (string, error) {
	return strings.ReplaceAll(uri, "mobile.twitter.com", "twitter.com"), nil
}
