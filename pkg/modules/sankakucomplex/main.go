package sankakucomplex

import (
	"fmt"
	"github.com/DaRealFreak/watcher-go/pkg/database"
	"github.com/DaRealFreak/watcher-go/pkg/http/session"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type sankakuComplex struct {
	models.Module
}

// generate new module and register uri schema
func NewModule(dbIO *database.DbIO, uriSchemas map[string][]*regexp.Regexp) *models.Module {
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
	module.RegisterUriSchema(uriSchemas)
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
func (m *sankakuComplex) IsLoggedIn() (LoggedIn bool) {
	return m.LoggedIn
}

// add our pattern to the uri schemas
func (m *sankakuComplex) RegisterUriSchema(uriSchemas map[string][]*regexp.Regexp) {
	var moduleUriSchemas []*regexp.Regexp
	test, _ := regexp.Compile(".*sankakucomplex.com")
	moduleUriSchemas = append(moduleUriSchemas, test)
	uriSchemas[m.Key()] = moduleUriSchemas
}

// login function
func (m *sankakuComplex) Login(account *models.Account) bool {
	values := url.Values{
		"url":            {""},
		"user[name]":     {account.Username},
		"user[password]": {account.Password},
		"commit":         {"Login"},
	}

	res, _ := m.post("https://chan.sankakucomplex.com/user/authenticate", values, 0)
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

// custom POST function to check for specific status codes and messages
func (m *sankakuComplex) post(uri string, data url.Values, tries int) (*http.Response, error) {
	res, err := m.Session.Post(uri, data)
	if err == nil && res.StatusCode == 429 {
		log.Info(fmt.Sprintf("too many requests, sleeping '%d' seconds", tries+1*60))
		time.Sleep(time.Duration(tries+1*60) * time.Second)
		return m.post(uri, data, tries+1)
	}
	return res, err
}

// custom GET function to check for specific status codes and messages
func (m *sankakuComplex) get(uri string, tries int) (*http.Response, error) {
	res, err := m.Session.Get(uri)
	if err == nil && res.StatusCode == 429 {
		log.Info(fmt.Sprintf("too many requests, sleeping '%d' seconds", tries+1*60))
		time.Sleep(time.Duration(tries+1*60) * time.Second)
		return m.get(uri, tries+1)
	}
	return res, err
}
