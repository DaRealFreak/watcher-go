// Package fantia contains the implementation of the fantia.jp module
package fantia

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


// fantia contains the implementation of the ModuleInterface
type fantia struct {
	*models.Module
	settings  fantiaSettings
	csrfToken string
}

type fantiaSettings struct{}

// nolint: gochecknoinits
// init function registers the bare module to the module factory
func init() {
	modules.GetModuleFactory().RegisterModule(NewBareModule())
}

// NewBareModule returns a bare module implementation for the CLI options
func NewBareModule() *models.Module {
	module := &models.Module{
		Key:           "fantia.jp",
		RequiresLogin: true,
		LoggedIn:      false,
		URISchemas: []*regexp.Regexp{
			regexp.MustCompile(`.*fantia\.jp`),
		},
	}
	module.ModuleInterface = &fantia{
		Module: module,
	}

	// register module to log formatter
	formatter.AddFieldMatchColorScheme("module", &formatter.FieldMatch{
		Value: module.Key,
		Color: "232:209",
	})

	return module
}

// InitializeModule initializes the module
func (m *fantia) InitializeModule() {
	raven.CheckError(viper.UnmarshalKey(
		fmt.Sprintf("Modules.%s", m.GetViperModuleKey()),
		&m.settings,
	))

	m.Session = tls_session.NewTlsClientSession(m.Key)
	raven.CheckError(m.Session.SetProxy(m.GetProxySettings()))
}

// AddModuleCommand adds custom module specific settings and commands to our application
func (m *fantia) AddModuleCommand(command *cobra.Command) {
	m.AddProxyCommands(command)
}

// SetCookies loads stored cookies and checks for the required _session_id
func (m *fantia) SetCookies() {
	m.Module.SetCookies()

	if cookie := m.DbIO.GetCookie("_session_id", m); cookie != nil {
		m.LoggedIn = true
	}
}

// Login logs us in for the current session if possible/account available
func (m *fantia) Login(_ *models.Account) bool {
	return m.LoggedIn
}

// Parse parses the tracked item
func (m *fantia) Parse(item *models.TrackedItem) error {
	if strings.Contains(item.URI, "/posts/") {
		return m.parsePost(item)
	}
	return m.parseFanclub(item)
}

// AddItem normalizes the URI before adding it to the database
func (m *fantia) AddItem(uri string) (string, error) {
	return strings.TrimRight(uri, "/"), nil
}
