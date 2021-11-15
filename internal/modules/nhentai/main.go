// Package nhentai contains the implementation of the nhentai module
package nhentai

import (
	"regexp"
	"strings"

	formatter "github.com/DaRealFreak/colored-nested-formatter"
	"github.com/DaRealFreak/watcher-go/internal/http/session"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/spf13/cobra"
)

type nhentai struct {
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
		Key:           "nhentai.net",
		RequiresLogin: false,
		LoggedIn:      false,
		URISchemas: []*regexp.Regexp{
			regexp.MustCompile(`.*nhentai.net`),
		},
		ProxyLoopIndex: -1,
	}
	module.ModuleInterface = &nhentai{
		Module: module,
	}

	// register module to log formatter
	formatter.AddFieldMatchColorScheme("module", &formatter.FieldMatch{
		Value: module.Key,
		Color: "255:197",
	})

	return module
}

// InitializeModule initializes the module
func (m *nhentai) InitializeModule() {
	m.Session = session.NewSession(m.Key)

	// set the proxy if requested
	raven.CheckError(m.Session.SetProxy(m.GetProxySettings()))
}

// AddSettingsCommand adds custom module specific settings and commands to our application
func (m *nhentai) AddSettingsCommand(command *cobra.Command) {
	m.AddProxyCommands(command)
	m.AddProxyLoopCommands(command)
}

// Login logs us in for the current session if possible/account available
func (m *nhentai) Login(account *models.Account) bool {
	return m.LoggedIn
}

// Parse parses the tracked item
func (m *nhentai) Parse(item *models.TrackedItem) error {
	if strings.Contains("/g/", item.URI) {
		return m.parseGallery(item)
	} else {

	}

	return nil
}
