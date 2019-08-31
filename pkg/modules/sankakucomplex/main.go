package sankakucomplex

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/DaRealFreak/watcher-go/cmd/log/formatter"
	"github.com/DaRealFreak/watcher-go/pkg/http/session"
	"github.com/DaRealFreak/watcher-go/pkg/models"
)

// sankakuComplex contains the implementation of the ModuleInterface
type sankakuComplex struct {
	models.Module
}

// NewModule generates new module and registers the URI schema
func NewModule(dbIO models.DatabaseInterface, uriSchemas map[string][]*regexp.Regexp) *models.Module {
	// register empty sub module to point to
	var subModule = sankakuComplex{}
	sankakuSession := session.NewSession()
	sankakuSession.ModuleKey = subModule.Key()

	// initialize the Module with the session/database and login status
	module := models.Module{
		DbIO:            dbIO,
		Session:         sankakuSession,
		LoggedIn:        false,
		ModuleInterface: &subModule,
	}
	// set the module implementation for access to the session, database, etc
	subModule.Module = module

	// register the uri schema
	module.RegisterURISchema(uriSchemas)
	return &module
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
	return m.LoggedIn
}

// Parse parses the tracked item
func (m *sankakuComplex) Parse(item *models.TrackedItem) {
	// ToDo: add book support
	downloadQueue := m.parseGallery(item)

	m.ProcessDownloadQueue(downloadQueue, item)
}

// init registers the module to the log formatter
func init() {
	formatter.AddFieldMatchColorScheme("module", &formatter.FieldMatch{
		Value: (&sankakuComplex{}).Key(),
		Color: "232:172",
	})
}
