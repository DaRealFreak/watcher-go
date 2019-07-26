package sankakucomplex

import (
	"fmt"
	"net/url"
	"regexp"
	"watcher-go/database"
	"watcher-go/http"
	"watcher-go/modules/template"
)

type SankakuComplex struct {
	session *http.Session
}

// generate new module and register uri schema
func NewModule(uriSchemas map[string][]*regexp.Regexp) *template.Module {
	var templateImplementation template.ModuleInterface = SankakuComplex{
		session: http.NewSession(),
	}

	module := template.Module{
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

func (m SankakuComplex) Login(account *database.Account) {
	values := url.Values{
		"url":            {""},
		"user[name]":     {account.Username},
		"user[password]": {account.Password},
		"commit":         {"Login"},
	}
	// ToDo: validation of login response
	_, _ = m.session.Post("https://chan.sankakucomplex.com/user/authenticate", values, 0)
}

func (m SankakuComplex) Parse(item *database.TrackedItem) {
	fmt.Println(item.Uri, item.CurrentItem)
	panic("implement me")
}
