// Package patreon contains the implementation of the patreon module
package patreon

import (
	"context"
	"os"
	"regexp"

	formatter "github.com/DaRealFreak/colored-nested-formatter"
	"github.com/DaRealFreak/watcher-go/pkg/http/session"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/DaRealFreak/watcher-go/pkg/modules"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

// patreon contains the implementation of the ModuleInterface
type patreon struct {
	*models.Module
}

// nolint: gochecknoinits
// init function registers the bare and the normal module to the module factories
func init() {
	modules.GetModuleFactory().RegisterModule(NewBareModule())
}

// NewBareModule returns a bare module implementation for the CLI options
func NewBareModule() *models.Module {
	module := &models.Module{
		Key:           "patreon.com",
		RequiresLogin: false,
		LoggedIn:      false,
		URISchemas: []*regexp.Regexp{
			regexp.MustCompile(".*patreon.com"),
		},
	}
	module.ModuleInterface = &patreon{
		Module: module,
	}

	// register module to log formatter
	formatter.AddFieldMatchColorScheme("module", &formatter.FieldMatch{
		Value: module.Key,
		Color: "232:167",
	})

	return module
}

// InitializeModule initializes the module
func (m *patreon) InitializeModule() {
	oAuthClient := m.DbIO.GetOAuthClient(m)
	if oAuthClient == nil || oAuthClient.AccessToken == "" {
		log.WithField("module", m.Key).Errorf(
			"module requires an OAuth2 access token, but no access token got found",
		)
		// Errorf will already exit with code 1, this line is just for the IDE
		os.Exit(1)
	}

	// initialize session
	m.Session = session.NewSession(m.Key)
	// set the proxy if requested
	raven.CheckError(m.Session.SetProxy(m.GetProxySettings()))

	// set context with own http client for OAuth2 library to use
	httpClientContext := context.WithValue(context.Background(), oauth2.HTTPClient, m.Session.GetClient())

	// create static token source and retrieve routed http client
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: oAuthClient.AccessToken})
	tc := oauth2.NewClient(httpClientContext, ts)

	m.Session.SetClient(tc)
}

// AddSettingsCommand adds custom module specific settings and commands to our application
func (m *patreon) AddSettingsCommand(command *cobra.Command) {
	m.AddProxyCommands(command)
}

// Login logs us in for the current session if possible/account available
func (m *patreon) Login(account *models.Account) bool {
	return m.LoggedIn
}

// Parse parses the tracked item
func (m *patreon) Parse(item *models.TrackedItem) error {
	return nil
}
