// Package sankakucomplex contains the implementation of the sankakucomplex module
package sankakucomplex

import (
	"regexp"

	"github.com/DaRealFreak/watcher-go/internal/modules/sankakucomplex/api"

	formatter "github.com/DaRealFreak/colored-nested-formatter"
	"github.com/DaRealFreak/watcher-go/internal/http/session"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/spf13/cobra"
)

// sankakuComplex contains the implementation of the ModuleInterface
type sankakuComplex struct {
	*models.Module
	api *api.SankakuComplexApi
}

// nolint: gochecknoinits
// init function registers the bare and the normal module to the module factories
func init() {
	modules.GetModuleFactory().RegisterModule(NewBareModule())
}

// NewBareModule returns a bare module implementation for the CLI options
func NewBareModule() *models.Module {
	module := &models.Module{
		Key:           "chan.sankakucomplex.com",
		RequiresLogin: false,
		LoggedIn:      false,
		URISchemas: []*regexp.Regexp{
			regexp.MustCompile(".*sankakucomplex.com"),
		},
	}
	module.ModuleInterface = &sankakuComplex{
		Module: module,
	}

	// register module to log formatter
	formatter.AddFieldMatchColorScheme("module", &formatter.FieldMatch{
		Value: module.Key,
		Color: "232:208",
	})

	return module
}

// InitializeModule initializes the module
func (m *sankakuComplex) InitializeModule() {
	// set the module implementation for access to the session, database, etc
	m.Session = session.NewSession(m.Key)

	// set the proxy if requested
	raven.CheckError(m.Session.SetProxy(m.GetProxySettings()))

	m.api = api.NewSankakuComplexApi(m.Key, m.Session, nil)
}

// AddModuleCommand adds custom module specific settings and commands to our application
func (m *sankakuComplex) AddModuleCommand(command *cobra.Command) {
	m.AddProxyCommands(command)
}

// Login logs us in for the current session if possible/account available
func (m *sankakuComplex) Login(account *models.Account) bool {
	// overwrite our previous API with a logged in instance
	m.api = api.NewSankakuComplexApi(m.Key, m.Session, account)

	return m.api.LoginSuccessful()
}

// Parse parses the tracked item
func (m *sankakuComplex) Parse(item *models.TrackedItem) error {
	// ToDo: add book support
	downloadQueue, err := m.parseGallery(item)
	if err != nil {
		return err
	}

	return m.processDownloadQueue(downloadQueue, item)
}
