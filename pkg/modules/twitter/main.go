// Package twitter contains the implementation of the twitter module
package twitter

import (
	"context"
	"os"
	"regexp"
	"time"

	formatter "github.com/DaRealFreak/colored-nested-formatter"
	"github.com/DaRealFreak/watcher-go/pkg/http/session"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/DaRealFreak/watcher-go/pkg/modules"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"golang.org/x/time/rate"
)

// twitter contains the implementation of the ModuleInterface
type twitter struct {
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

	// initialize session
	twitterSession := session.NewSession(m.Key)
	// twitter rate limits: 900 requests per 15 minutes
	twitterSession.RateLimiter = rate.NewLimiter(rate.Every(1*time.Second), 900)
	m.Session = twitterSession

	// set the proxy if requested
	raven.CheckError(m.Session.SetProxy(m.GetProxySettings()))

	config := &clientcredentials.Config{
		ClientID:     oauthClient.ClientID,
		ClientSecret: oauthClient.ClientSecret,
		TokenURL:     "https://api.twitter.com/oauth2/token",
	}

	// set context with own http client for OAuth2 library to use
	httpClientContext := context.WithValue(context.Background(), oauth2.HTTPClient, m.Session.GetClient())

	// add OAuth2 round tripper from our client credentials context
	m.Session.SetClient(config.Client(httpClientContext))
}

// AddSettingsCommand adds custom module specific settings and commands to our application
func (m *twitter) AddSettingsCommand(command *cobra.Command) {
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
