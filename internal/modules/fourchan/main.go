// Package fourchan contains the implementation of the 4chan module
package fourchan

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

// fourChan contains the implementation of the ModuleInterface
type fourChan struct {
	*models.Module
	threadPattern *regexp.Regexp
}

// init function registers the bare and the normal module to the module factories
func init() {
	modules.GetModuleFactory().RegisterModule(NewBareModule())
}

// NewBareModule returns a bare module implementation for the CLI options
func NewBareModule() *models.Module {
	module := &models.Module{
		Key:           "4chan.org",
		RequiresLogin: false,
		LoggedIn:      false,
		URISchemas: []*regexp.Regexp{
			regexp.MustCompile("4chan.org"),
			regexp.MustCompile("desuarchive.org"),
		},
	}
	module.ModuleInterface = &fourChan{
		Module:        module,
		threadPattern: regexp.MustCompile(`.*/(?P<BoardId>.*)/thread/(?P<ThreadID>.*)/`),
	}

	// register module to log formatter
	formatter.AddFieldMatchColorScheme("module", &formatter.FieldMatch{
		Value: module.Key,
		Color: "231:88",
	})

	return module
}

// InitializeModule initializes the module
func (m *fourChan) InitializeModule() {
	// set the module implementation for access to the session, database, etc
	m.Session = session.NewSession(m.Key)

	// set the proxy if requested
	raven.CheckError(m.Session.SetProxy(m.GetProxySettings()))
}

// AddModuleCommand adds custom module specific settings and commands to our application
func (m *fourChan) AddModuleCommand(command *cobra.Command) {
	m.AddProxyCommands(command)
}

// Login logs us in for the current session if possible/account available
func (m *fourChan) Login(_ *models.Account) bool {
	return true
}

// Parse parses the tracked item
func (m *fourChan) Parse(item *models.TrackedItem) error {
	if strings.Contains(item.URI, "/thread/") {
		return m.parseThread(item)
	} else if strings.Contains(item.URI, "/search/") {
		return m.parseSearch(item)
	}

	return nil
}
