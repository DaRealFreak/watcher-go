package sankakucomplex

import (
	"fmt"
	"github.com/golang/glog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
	"watcher-go/cmd/watcher/http_wrapper"
	"watcher-go/cmd/watcher/models"
)

type SankakuComplex struct {
	session  *http_wrapper.Session
	loggedIn bool
}

// generate new module and register uri schema
func NewModule(uriSchemas map[string][]*regexp.Regexp) *models.Module {
	var subModule = SankakuComplex{
		session:  http_wrapper.NewSession(),
		loggedIn: false,
	}
	var templateImplementation models.ModuleInterface = &subModule

	module := models.Module{
		Module: templateImplementation,
	}
	// register the uri schema
	module.Module.RegisterUriSchema(uriSchemas)
	return &module
}

// retrieve the module key
func (m *SankakuComplex) Key() (key string) {
	return "chan.sankakucomplex.com"
}

// retrieve the logged in status
func (m *SankakuComplex) IsLoggedIn() (loggedIn bool) {
	return m.loggedIn
}

// add our pattern to the uri schemas
func (m *SankakuComplex) RegisterUriSchema(uriSchemas map[string][]*regexp.Regexp) {
	var moduleUriSchemas []*regexp.Regexp
	test, _ := regexp.Compile(".*.sankakucomplex.com")
	moduleUriSchemas = append(moduleUriSchemas, test)
	uriSchemas[m.Key()] = moduleUriSchemas
}

// login function
func (m *SankakuComplex) Login(account *models.Account) bool {
	values := url.Values{
		"url":            {""},
		"user[name]":     {account.Username},
		"user[password]": {account.Password},
		"commit":         {"Login"},
	}

	res, _ := m.Post("https://chan.sankakucomplex.com/user/authenticate", values, 0)
	htmlResponse, _ := m.session.GetDocument(res).Html()
	m.loggedIn = strings.Contains(htmlResponse, "You are now logged in")
	return m.loggedIn
}

func (m *SankakuComplex) Parse(item *models.TrackedItem) {
	panic("implement me")
}

// custom Post function to check for specific status codes and messages
func (m *SankakuComplex) Post(url string, data url.Values, tries int) (*http.Response, error) {
	res, err := m.session.Post("https://chan.sankakucomplex.com/user/authenticate", data, tries)
	if err == nil && res.StatusCode == 429 {
		glog.Info(fmt.Sprintf("too many requests, sleeping '%d' seconds", tries+1*60))
		time.Sleep(time.Duration(tries+1*60) * time.Second)
		return m.Post(url, data, tries+1)
	}
	return res, err
}
