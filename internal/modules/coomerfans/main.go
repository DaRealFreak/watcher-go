// Package coomerfans contains the implementation of the coomerfans.com module
package coomerfans

import (
	"fmt"
	"regexp"
	"time"

	formatter "github.com/DaRealFreak/colored-nested-formatter/v2"
	"github.com/DaRealFreak/watcher-go/internal/http/tls_session"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/time/rate"
)

// baseURL is the canonical host for all requests.
const baseURL = "https://coomerfans.com"

type coomerfans struct {
	*models.Module
	settings coomerfansSettings
}

type coomerfansSettings struct {
	RateLimit *int `mapstructure:"rate_limit"`
}

// nolint: gochecknoinits
// init registers the bare module to the module factory.
func init() {
	modules.GetModuleFactory().RegisterModule(NewBareModule())
}

// NewBareModule returns a bare module implementation for the CLI options.
func NewBareModule() *models.Module {
	module := &models.Module{
		Key:           "coomerfans.com",
		RequiresLogin: false,
		LoggedIn:      false,
		URISchemas: []*regexp.Regexp{
			regexp.MustCompile(`coomerfans\.com`),
		},
		SettingsSchema: coomerfansSettings{},
	}
	module.ModuleInterface = &coomerfans{
		Module: module,
	}

	// register module to log formatter
	formatter.AddFieldMatchColorScheme("module", &formatter.FieldMatch{
		Value: module.Key,
		Color: "232:213",
	})

	return module
}

// InitializeModule initializes the module.
func (m *coomerfans) InitializeModule() {
	raven.CheckError(viper.UnmarshalKey(
		fmt.Sprintf("Modules.%s", m.GetViperModuleKey()),
		&m.settings,
	))

	rateLimit := 2500
	if m.settings.RateLimit != nil {
		rateLimit = *m.settings.RateLimit
	}

	session := tls_session.NewTlsClientSession(m.Key)
	session.RateLimiter = rate.NewLimiter(rate.Every(time.Duration(rateLimit)*time.Millisecond), 1)
	m.Session = session

	// set the proxy if requested
	raven.CheckError(m.Session.SetProxy(m.GetProxySettings()))
}

// AddModuleCommand adds custom module specific settings and commands.
func (m *coomerfans) AddModuleCommand(command *cobra.Command) {
	m.AddProxyCommands(command)
}

// Login logs us in for the current session if possible/account available.
func (m *coomerfans) Login(_ *models.Account) bool {
	return true
}

// Parse parses the tracked item.
func (m *coomerfans) Parse(item *models.TrackedItem) error {
	if postURLPattern.MatchString(item.URI) {
		return m.parsePost(item)
	}
	return m.parseUser(item)
}
