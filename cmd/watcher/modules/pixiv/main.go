package pixiv

import (
	"regexp"
	"watcher-go/cmd/watcher/database"
	"watcher-go/cmd/watcher/http_wrapper"
	"watcher-go/cmd/watcher/models"
)

type pixiv struct {
	models.Module
}

// generate new module and register uri schema
func NewModule(dbIO *database.DbIO, uriSchemas map[string][]*regexp.Regexp) *models.Module {
	// register empty sub module to point to
	var subModule = pixiv{}

	// initialize the Module with the session/database and login status
	module := models.Module{
		DbIO:            dbIO,
		Session:         http_wrapper.NewSession(),
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
func (m *pixiv) Key() (key string) {
	return "pixiv.net"
}

// retrieve the logged in status
func (m *pixiv) IsLoggedIn() (LoggedIn bool) {
	return m.LoggedIn
}

// add our pattern to the uri schemas
func (m *pixiv) RegisterUriSchema(uriSchemas map[string][]*regexp.Regexp) {
	var moduleUriSchemas []*regexp.Regexp
	test, _ := regexp.Compile(".*.pixiv.(co.jp|net)")
	moduleUriSchemas = append(moduleUriSchemas, test)
	uriSchemas[m.Key()] = moduleUriSchemas
}

// login function
func (m *pixiv) Login(account *models.Account) bool {
	return false
}

func (m *pixiv) Parse(item *models.TrackedItem) {
}
