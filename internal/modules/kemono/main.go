package kemono

import (
	"regexp"

	cloudflarebp "github.com/DaRealFreak/cloudflare-bp-go"
	formatter "github.com/DaRealFreak/colored-nested-formatter"
	"github.com/DaRealFreak/watcher-go/internal/http/session"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/spf13/cobra"
)

type kemono struct {
	*models.Module
}

// init function registers the bare and the normal module to the module factories
func init() {
	modules.GetModuleFactory().RegisterModule(NewBareModule())
}

// NewBareModule returns a bare module implementation for the CLI options
func NewBareModule() *models.Module {
	module := &models.Module{
		Key:           "kemono.party",
		RequiresLogin: false,
		LoggedIn:      false,
		URISchemas: []*regexp.Regexp{
			regexp.MustCompile(`kemono.party`),
		},
	}
	module.ModuleInterface = &kemono{
		Module: module,
	}

	// register module to log formatter
	formatter.AddFieldMatchColorScheme("module", &formatter.FieldMatch{
		Value: module.Key,
		Color: "208:39",
	})

	return module
}

// InitializeModule initializes the module
func (m *kemono) InitializeModule() {
	// set the module implementation for access to the session, database, etc
	m.Session = session.NewSession(m.Key)

	// set the proxy if requested
	raven.CheckError(m.Session.SetProxy(m.GetProxySettings()))
}

// AddModuleCommand adds custom module specific settings and commands to our application
func (m *kemono) AddModuleCommand(command *cobra.Command) {
	m.AddProxyCommands(command)
}

// Login logs us in for the current session if possible/account available
func (m *kemono) Login(_ *models.Account) bool {
	return true
}

// Parse parses the tracked item
func (m *kemono) Parse(item *models.TrackedItem) error {
	return nil
}

// AddRoundTrippers adds the round trippers for CloudFlare, adds a custom user agent
// and implements the implicit OAuth2 authentication and sets the Token round tripper
func (m *kemono) addRoundTrippers() {
	client := m.Session.GetClient()
	// apply CloudFlare bypass
	options := cloudflarebp.GetDefaultOptions()
	client.Transport = cloudflarebp.AddCloudFlareByPass(client.Transport, options)
}
