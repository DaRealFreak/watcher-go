// Package jinjamodoki contains the implementation of the jinjamodoki module
package jinjamodoki

import (
	"net/url"
	"regexp"
	"time"

	formatter "github.com/DaRealFreak/colored-nested-formatter"
	"github.com/DaRealFreak/watcher-go/pkg/http/session"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/DaRealFreak/watcher-go/pkg/modules"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
	"github.com/spf13/cobra"
	"golang.org/x/time/rate"
)

// jinjaModoki contains the implementation of the ModuleInterface
type jinjaModoki struct {
	*models.Module
	baseURL *url.URL
}

// nolint: gochecknoinits
// init function registers the bare and the normal module to the module factories
func init() {
	modules.GetModuleFactory().RegisterModule(NewBareModule())
}

// NewBareModule returns a bare module implementation for the CLI options
func NewBareModule() *models.Module {
	module := &models.Module{
		Key:           "jinja-modoki.com",
		RequiresLogin: false,
		LoggedIn:      false,
		URISchemas: []*regexp.Regexp{
			regexp.MustCompile("jinja-modoki.com"),
		},
	}
	module.ModuleInterface = &jinjaModoki{
		Module: module,
	}

	// register module to log formatter
	formatter.AddFieldMatchColorScheme("module", &formatter.FieldMatch{
		Value: module.Key,
		Color: "232:231",
	})

	return module
}

// InitializeModule initializes the module
func (m *jinjaModoki) InitializeModule() {
	m.baseURL, _ = url.Parse("https://gs-uploader.jinja-modoki.com/")

	moduleSession := session.NewSession(m.Key)
	moduleSession.RateLimiter = rate.NewLimiter(rate.Every(1*time.Second), 1)
	m.Session = moduleSession
	// set the proxy if requested
	raven.CheckError(m.Session.SetProxy(m.GetProxySettings()))

	client := m.Session.GetClient()
	client.Transport = m.SetReferer(client.Transport)

	// disable browsing access restrictions
	_, err := m.Session.Post("https://gs-uploader.jinja-modoki.com/upld-index.php?", url.Values{
		"mode":          {"complete"},
		"prev_mode":     {"top"},
		"item":          {"restriction"},
		"restriction[]": {"0", "1", "2", "3", "4"},
	})
	raven.CheckError(err)
}

// AddSettingsCommand adds custom module specific settings and commands to our application
func (m *jinjaModoki) AddSettingsCommand(command *cobra.Command) {
	m.AddProxyCommands(command)
}

// Login logs us in for the current session if possible/account available
func (m *jinjaModoki) Login(account *models.Account) bool {
	return m.LoggedIn
}

// Parse parses the tracked item
func (m *jinjaModoki) Parse(item *models.TrackedItem) error {
	return m.parsePage(item)
}
