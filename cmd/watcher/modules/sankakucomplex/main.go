package sankakucomplex

import (
	"fmt"
	"net/url"
	"regexp"
	"watcher-go/cmd/watcher/models"
	"watcher-go/http"
)

type SankakuComplex struct {
	session *http.Session
}

// generate new module and register uri schema
func NewModule(uriSchemas map[string][]*regexp.Regexp) *models.Module {
	var templateImplementation models.ModuleInterface = SankakuComplex{
		session: http.NewSession(),
	}

	module := models.Module{
		Module: templateImplementation,
	}
	// register the uri schema
	module.Module.RegisterUriSchema(uriSchemas)
	return &module
}

// retrieve the module key
func (m SankakuComplex) Key() (key string) {
	return "chan.sankakucomplex.com"
}

// add our pattern to the uri schemas
func (m SankakuComplex) RegisterUriSchema(uriSchemas map[string][]*regexp.Regexp) {
	var moduleUriSchemas []*regexp.Regexp
	test, _ := regexp.Compile(".*.sankakucomplex.com")
	moduleUriSchemas = append(moduleUriSchemas, test)
	uriSchemas[m.Key()] = moduleUriSchemas
}

func (m SankakuComplex) Login(account *models.Account) {
	values := url.Values{
		"url":            {""},
		"user[name]":     {account.Username},
		"user[password]": {account.Password},
		"commit":         {"Login"},
	}
	// ToDo: validation of login response
	_, _ = m.session.Post("https://chan.sankakucomplex.com/user/authenticate", values, 0)
}

func (m SankakuComplex) Parse(item *models.TrackedItem) {
	fmt.Println(item.Uri, item.CurrentItem)
	panic("implement me")
}
