package deviantart

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/DaRealFreak/watcher-go/cmd/log/formatter"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/DaRealFreak/watcher-go/pkg/modules/deviantart/session"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
	"github.com/PuerkitoBio/goquery"
	log "github.com/sirupsen/logrus"
)

// deviantArt contains the implementation of the ModuleInterface
type deviantArt struct {
	models.Module
	deviantArtSession  *session.DeviantArtSession
	userGalleryPattern *regexp.Regexp
}

// NewModule generates new module and registers the URI schema
func NewModule(dbIO models.DatabaseInterface, uriSchemas map[string][]*regexp.Regexp) *models.Module {
	// register empty sub module to point to
	var subModule = deviantArt{
		deviantArtSession: session.NewSession(),
		userGalleryPattern: regexp.MustCompile(
			`https://www\.deviantart\.com/([^/?&]+)(/gallery((/|/\?catpath=/)?))?$`,
		),
	}

	// initialize the Module with the session/database and login status
	module := models.Module{
		DbIO:            dbIO,
		Session:         subModule.deviantArtSession,
		LoggedIn:        false,
		ModuleInterface: &subModule,
	}
	// set the module implementation for access to the session, database, etc
	subModule.Module = module
	subModule.deviantArtSession.ModuleKey = subModule.Key()

	// register the uri schema
	module.RegisterURISchema(uriSchemas)
	// register module to log formatter
	formatter.AddFieldMatchColorScheme("module", &formatter.FieldMatch{
		Value: module.Key(),
		Color: "232:34",
	})
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
	uriSchemas[m.Key()] = []*regexp.Regexp{
		regexp.MustCompile(".*deviantart.com"),
		regexp.MustCompile(`DeviantArt://.*`),
	}
}

// Login logs us in for the current session if possible/account available
func (m *deviantArt) Login(account *models.Account) bool {
	if !m.prepareSessionForOAuth2(account) {
		log.WithField("module", m.Key()).Warning(
			"preparing session for OAuth2 Token generation failed, please check your account",
		)
		return false
	}

	// call the utility endpoint function placebo to check the validity of the generated token
	placebo, err := m.Placebo()
	// check placebo response if the token can be used
	m.LoggedIn = err == nil && placebo.Status == "success"
	return m.LoggedIn
}

// prepareSessionForOAuth2 is used if the OAuth2 step should be completed automatically
// so we log in into the website with the session before retrieving the OAuth2 Code
// if login fails we use the browser solution as fallback which wouldn't even require a user in the database
func (m *deviantArt) prepareSessionForOAuth2(account *models.Account) bool {
	res, err := m.Session.Get("https://www.deviantart.com/users/login")
	raven.CheckError(err)

	info := m.getLoginCSRFToken(res)
	if !(info.CSRFToken != "") {
		raven.CheckError(fmt.Errorf("could not retrieve CSRF token from login page"))
	}

	values := url.Values{
		"referer":    {info.Referer},
		"csrf_token": {info.CSRFToken},
		"challenge":  {"0"},
		"username":   {account.Username},
		"password":   {account.Password},
		"remember":   {"on"},
	}
	res, _ = m.Session.Post("https://www.deviantart.com/_sisu/do/signin", values)
	doc := m.Session.GetDocument(res)
	html, err := doc.Html()
	raven.CheckError(err)
	return strings.Contains(html, "\"loggedIn\":true")
}

// getLoginCSRFToken returns the CSRF token from the login site to use in our POST login request
func (m *deviantArt) getLoginCSRFToken(res *http.Response) (loginInfo loginInfo) {
	jsonPattern := regexp.MustCompile(`JSON.parse\((?P<Number>.*csrfToken.*?)\);`)
	doc := m.Session.GetDocument(res)
	scriptTags := doc.Find("script")
	scriptTags.Each(func(row int, selection *goquery.Selection) {
		// no need for further checks if we already have our login info
		if loginInfo.CSRFToken != "" {
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
	// special behaviour: viewing "all" gallery doesn't contain unique gallery id (using featured uuid)
	// so it won't retrieve all items, only the featured ones
	// we use the /gallery/all API endpoint in that case to retrieve actually all deviations
	if m.userGalleryPattern.MatchString(item.URI) {
		m.parseGalleryAll(item)
		return
	}

	appURL := item.URI
	if !strings.HasPrefix(appURL, "DeviantArt://") {
		var exists bool
		appURL, exists = m.getAppURL(item.URI)
		if !exists {
			log.WithField("module", m.Key()).Warnf("couldn't extract app url from page %s", item.URI)
			// couldn't extract url from passed uri
			return
		}
	}
	switch {
	case strings.HasPrefix(appURL, "DeviantArt://collection/"):
		m.parseCollection(appURL, item)
	case strings.HasPrefix(appURL, "DeviantArt://tag/"):
		m.parseTag(appURL, item)
	case strings.HasPrefix(appURL, "DeviantArt://deviation/"):
		m.parseDeviation(appURL, item)
	case strings.HasPrefix(appURL, "DeviantArt://gallery/"):
		m.parseGallery(appURL, item)
	case strings.HasPrefix(appURL, "DeviantArt://watchfeed"):
		// ToDo: debug android client, API documentation is completely wrong (even wrong response)
		// and pagination won't work either
		m.parseFeed(item)
	}
}

// getAppURL extracts the meta attribute da:appurl and returns it
func (m *deviantArt) getAppURL(uri string) (appURL string, exists bool) {
	res, err := m.Session.Get(uri)
	raven.CheckError(err)

	doc := m.Session.GetDocument(res)
	return doc.Find("meta[property=\"da:appurl\"]").First().Attr("content")
}
