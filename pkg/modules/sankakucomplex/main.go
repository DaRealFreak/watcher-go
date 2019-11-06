// Package sankakucomplex contains the implementation of the sankakucomplex module
package sankakucomplex

import (
	"net/url"
	"regexp"
	"strings"

	formatter "github.com/DaRealFreak/colored-nested-formatter"
	"github.com/DaRealFreak/watcher-go/pkg/http/session"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/DaRealFreak/watcher-go/pkg/modules"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
	"github.com/spf13/cobra"
)

// sankakuComplex contains the implementation of the ModuleInterface
type sankakuComplex struct {
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
		LoggedIn: false,
	}
	module.ModuleInterface = &sankakuComplex{
		Module: module,
	}

	// register module to log formatter
	formatter.AddFieldMatchColorScheme("module", &formatter.FieldMatch{
		Value: module.Key(),
		Color: "232:208",
	})

	return module
}

// InitializeModule initializes the module
func (m *sankakuComplex) InitializeModule() {
	// set the module implementation for access to the session, database, etc
	m.Session = session.NewSession(m.Key())

	// set the proxy if requested
	raven.CheckError(m.Session.SetProxy(m.GetProxySettings()))
}

// Key returns the module key
func (m *sankakuComplex) Key() (key string) {
	return "chan.sankakucomplex.com"
}

// RequiresLogin checks if this module requires a login to work
func (m *sankakuComplex) RequiresLogin() (requiresLogin bool) {
	return false
}

// IsLoggedIn returns the logged in status
func (m *sankakuComplex) IsLoggedIn() bool {
	return m.LoggedIn
}

// RegisterURISchema adds our pattern to the URI Schemas
func (m *sankakuComplex) RegisterURISchema(uriSchemas map[string][]*regexp.Regexp) {
	uriSchemas[m.Key()] = []*regexp.Regexp{
		regexp.MustCompile(".*sankakucomplex.com"),
	}
}

// AddSettingsCommand adds custom module specific settings and commands to our application
func (m *sankakuComplex) AddSettingsCommand(command *cobra.Command) {
	m.AddProxyCommands(command)
}

// Login logs us in for the current session if possible/account available
func (m *sankakuComplex) Login(account *models.Account) bool {
	values := url.Values{
		"url":            {""},
		"user[name]":     {account.Username},
		"user[password]": {account.Password},
		"commit":         {"Login"},
	}

	res, _ := m.Session.Post("https://chan.sankakucomplex.com/user/authenticate", values)
	htmlResponse, _ := m.Session.GetDocument(res).Html()
	m.LoggedIn = strings.Contains(htmlResponse, "You are now logged in")
	m.TriedLogin = true

	return m.LoggedIn
}

// Parse parses the tracked item
func (m *sankakuComplex) Parse(item *models.TrackedItem) error {
	// ToDo: add book support
	downloadQueue, err := m.parseGallery(item)
	if err != nil {
		return err
	}

	return m.ProcessDownloadQueue(downloadQueue, item)
}
