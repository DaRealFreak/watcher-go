package sankakucomplex

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/DaRealFreak/watcher-go/pkg/http/session"
	"github.com/DaRealFreak/watcher-go/pkg/models"
)

type sankakuComplex struct {
	models.Module
}

// generate new module and register uri schema
func NewModule(dbIO models.DatabaseInterface, uriSchemas map[string][]*regexp.Regexp) *models.Module {
	// register empty sub module to point to
	var subModule = sankakuComplex{}

	// initialize the Module with the session/database and login status
	module := models.Module{
		DbIO:            dbIO,
		Session:         session.NewSession(),
		LoggedIn:        false,
		ModuleInterface: &subModule,
	}
	// set the module implementation for access to the session, database, etc
	subModule.Module = module
	// register the uri schema
	module.RegisterURISchema(uriSchemas)
	return &module
}

// retrieve the module key
func (m *sankakuComplex) Key() (key string) {
	return "chan.sankakucomplex.com"
}

// check if this module requires a login to work
func (m *sankakuComplex) RequiresLogin() (requiresLogin bool) {
	return false
}

// retrieve the logged in status
func (m *sankakuComplex) IsLoggedIn() bool {
	return m.LoggedIn
}

// add our pattern to the uri schemas
func (m *sankakuComplex) RegisterURISchema(uriSchemas map[string][]*regexp.Regexp) {
	var moduleURISchemas []*regexp.Regexp
	moduleURISchema := regexp.MustCompile(".*sankakucomplex.com")
	moduleURISchemas = append(moduleURISchemas, moduleURISchema)
	uriSchemas[m.Key()] = moduleURISchemas
}

// login function
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

// main functionality to parse and process the passed item
func (m *sankakuComplex) Parse(item *models.TrackedItem) {
	// ToDo: add book support
	downloadQueue := m.parseGallery(item)

	m.ProcessDownloadQueue(downloadQueue, item)
}
