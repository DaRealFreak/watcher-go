package ehentai

import (
	"net/url"
	"regexp"
	"strings"
	"watcher-go/pkg/database"
	"watcher-go/pkg/http_wrapper"
	"watcher-go/pkg/models"
)

type ehentai struct {
	models.Module
	galleryImageIdPattern    *regexp.Regexp
	galleryImageIndexPattern *regexp.Regexp
	searchGalleryIdPattern   *regexp.Regexp
}

// generate new module and register uri schema
func NewModule(dbIO *database.DbIO, uriSchemas map[string][]*regexp.Regexp) *models.Module {
	// register empty sub module to point to
	var subModule = ehentai{
		galleryImageIdPattern:    regexp.MustCompile("(\\w+-\\d+)"),
		galleryImageIndexPattern: regexp.MustCompile("\\w+-(?P<Number>\\d+)"),
		searchGalleryIdPattern:   regexp.MustCompile("(\\d+/\\w+)"),
	}

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
func (m *ehentai) Key() (key string) {
	return "e-hentai.com"
}

// check if this module requires a login to work
func (m *ehentai) RequiresLogin() (requiresLogin bool) {
	return false
}

// retrieve the logged in status
func (m *ehentai) IsLoggedIn() (LoggedIn bool) {
	return m.LoggedIn
}

// add our pattern to the uri schemas
func (m *ehentai) RegisterUriSchema(uriSchemas map[string][]*regexp.Regexp) {
	var moduleUriSchemas []*regexp.Regexp
	schema, _ := regexp.Compile(".*e[\\-x]hentai.org")
	moduleUriSchemas = append(moduleUriSchemas, schema)
	uriSchemas[m.Key()] = moduleUriSchemas
}

// login function
func (m *ehentai) Login(account *models.Account) bool {
	values := url.Values{
		"CookieDate":       {"1"},
		"b":                {"d"},
		"bt":               {"1-1"},
		"UserName":         {account.Username},
		"PassWord":         {account.Password},
		"ipb_login_submit": {"Login!"},
	}

	res, _ := m.Session.Post("https://forums.e-hentai.org/index.php?act=Login&CODE=01", values, 0)
	htmlResponse, _ := m.Session.GetDocument(res).Html()
	m.LoggedIn = strings.Contains(htmlResponse, "You are now logged in")

	// copy the cookies for e-hentai to exhentai
	ehUrl, _ := url.Parse("https://e-hentai.org")
	exUrl, _ := url.Parse("https://exhentai.org")
	m.Session.Client.Jar.SetCookies(exUrl, m.Session.Client.Jar.Cookies(ehUrl))

	return m.LoggedIn
}

func (m *ehentai) Parse(item *models.TrackedItem) {
	if strings.Contains(item.Uri, "/g/") {
		m.parseGallery(item)
	} else if strings.Contains(item.Uri, "/tag/") {
		m.parseSearch(item)
	}
}
