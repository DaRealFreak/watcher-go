// Package skeb contains the implementation of the skeb.jp module
package skeb

import (
	"fmt"
	"regexp"
	"strings"

	formatter "github.com/DaRealFreak/colored-nested-formatter/v2"
	"github.com/DaRealFreak/watcher-go/internal/http/tls_session"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// skeb contains the implementation of the ModuleInterface
type skeb struct {
	*models.Module
	settings skebSettings
}

type skebSettings struct {
	Role string `mapstructure:"role"`
}

// nolint: gochecknoinits
// init function registers the bare module to the module factory
func init() {
	modules.GetModuleFactory().RegisterModule(NewBareModule())
}

// NewBareModule returns a bare module implementation for the CLI options
func NewBareModule() *models.Module {
	module := &models.Module{
		Key:           "skeb.jp",
		RequiresLogin: false,
		LoggedIn:      false,
		URISchemas: []*regexp.Regexp{
			regexp.MustCompile(`.*skeb\.jp`),
		},
	}
	module.ModuleInterface = &skeb{
		Module: module,
	}

	// register module to log formatter
	formatter.AddFieldMatchColorScheme("module", &formatter.FieldMatch{
		Value: module.Key,
		Color: "232:214",
	})

	return module
}

// InitializeModule initializes the module
func (m *skeb) InitializeModule() {
	raven.CheckError(viper.UnmarshalKey(
		fmt.Sprintf("Modules.%s", m.GetViperModuleKey()),
		&m.settings,
	))

	if m.settings.Role == "" {
		m.settings.Role = "creator"
	}

	m.Session = tls_session.NewTlsClientSession(m.Key)
	raven.CheckError(m.Session.SetProxy(m.GetProxySettings()))
}

// AddModuleCommand adds custom module specific settings and commands to our application
func (m *skeb) AddModuleCommand(command *cobra.Command) {
	m.AddProxyCommands(command)
}

// Login logs us in for the current session if possible/account available
func (m *skeb) Login(_ *models.Account) bool {
	return false
}

// Parse parses the tracked item
func (m *skeb) Parse(item *models.TrackedItem) error {
	if strings.Contains(item.URI, "/works/") {
		return m.parseWork(item)
	}
	return m.parseProfile(item)
}

// AddItem normalizes the URI before adding it to the database
func (m *skeb) AddItem(uri string) (string, error) {
	return strings.TrimRight(uri, "/"), nil
}
