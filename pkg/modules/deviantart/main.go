package deviantart

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/DaRealFreak/watcher-go/pkg/http/session"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
	"github.com/PuerkitoBio/goquery"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

// deviantArt contains the implementation of the ModuleInterface
type deviantArt struct {
	models.Module
	token *oauth2.Token
}

// NewModule generates new module and registers the URI schema
func NewModule(dbIO models.DatabaseInterface, uriSchemas map[string][]*regexp.Regexp) *models.Module {
	// register empty sub module to point to
	var subModule = deviantArt{}

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

// Key returns the module key
func (m *deviantArt) Key() (key string) {
	return "deviantart.com"
}

// RequiresLogin checks if this module requires a login to work
func (m *deviantArt) RequiresLogin() (requiresLogin bool) {
	return true
}

// IsLoggedIn returns the logged in status
func (m *deviantArt) IsLoggedIn() bool {
	return m.LoggedIn
}

// RegisterURISchema adds our pattern to the URI Schemas
func (m *deviantArt) RegisterURISchema(uriSchemas map[string][]*regexp.Regexp) {
	var moduleURISchemas []*regexp.Regexp
	moduleURISchema := regexp.MustCompile(".*deviantart.com")
	moduleURISchemas = append(moduleURISchemas, moduleURISchema)
	uriSchemas[m.Key()] = moduleURISchemas
}

// Login logs us in for the current session if possible/account available
func (m *deviantArt) Login(account *models.Account) bool {
	m.prepareSessionForOAuth2(account)
	m.token = m.retrieveOAuth2Code()
	m.LoggedIn = m.token != nil
	return m.LoggedIn
}

// prepareSessionForOAuth2 is used if the OAuth2 step should be completed automatically
// so we log in into the website with the session before retrieving the OAuth2 Code
// if login fails we use the browser solution as fallback which wouldn't even require a user in the database
func (m *deviantArt) prepareSessionForOAuth2(account *models.Account) {
	res, err := m.Session.Get("https://www.deviantart.com/users/login")
	raven.CheckError(err)

	info := m.getLoginCSRFToken(res)
	if !(info.CsrfToken != "") {
		raven.CheckError(fmt.Errorf("could not retrieve CSRF token from login page"))
	}

	values := url.Values{
		"referer":    {info.Referer},
		"csrf_token": {info.CsrfToken},
		"challenge":  {"0"},
		"username":   {account.Username},
		"password":   {account.Password},
		"remember":   {"on"},
	}
	res, _ = m.Session.Post("https://www.deviantart.com/_sisu/do/signin", values)
	doc := m.Session.GetDocument(res)
	html, err := doc.Html()
	raven.CheckError(err)
	// login not successful
	if !strings.Contains(html, "\"loggedIn\":true") {
		log.Warning("login not successful, using browser authentication process as fallback")
	}
}

// getLoginCSRFToken returns the CSRF token from the login site to use in our POST login request
func (m *deviantArt) getLoginCSRFToken(res *http.Response) (loginInfo loginInfo) {
	jsonPattern := regexp.MustCompile(`JSON.parse\((?P<Number>.*csrfToken.*?)\);`)
	doc := m.Session.GetDocument(res)
	scriptTags := doc.Find("script")
	scriptTags.Each(func(row int, selection *goquery.Selection) {
		// no need for further checks if we already have our login info
		if loginInfo.CsrfToken != "" {
			return
		}

		scriptContent := selection.Text()
		if jsonPattern.MatchString(scriptContent) {
			s, err := strconv.Unquote(jsonPattern.FindStringSubmatch(scriptContent)[1])
			raven.CheckError(err)
			err = json.Unmarshal([]byte(s), &loginInfo)
			raven.CheckError(err)
		}
	})
	return loginInfo
}

// Parse parses the tracked item
func (m *deviantArt) Parse(item *models.TrackedItem) {
	values := url.Values{
		"access_token": {m.token.AccessToken},
	}
	res, _ := m.Session.Post("https://www.deviantart.com/api/v1/oauth2/placebo", values)
	fmt.Println(m.Session.GetDocument(res).Html())
	fmt.Println(item)
}
