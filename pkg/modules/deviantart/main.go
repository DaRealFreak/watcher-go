// Package deviantart contains the implementation of the deviantart module
package deviantart

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	formatter "github.com/DaRealFreak/colored-nested-formatter"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/DaRealFreak/watcher-go/pkg/modules"
	"github.com/DaRealFreak/watcher-go/pkg/modules/deviantart/session"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
	"github.com/PuerkitoBio/goquery"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// deviantArt contains the implementation of the ModuleInterface
type deviantArt struct {
	*models.Module
	deviantArtSession  *session.DeviantArtSession
	userGalleryPattern *regexp.Regexp
}

// nolint: gochecknoinits
// init function registers the bare and the normal module to the module factories
func init() {
	modules.GetModuleFactory().RegisterModule(NewBareModule())
}

// NewBareModule returns a bare module implementation for the CLI options
func NewBareModule() *models.Module {
	module := &models.Module{
		LoggedIn: true,
	}
	module.ModuleInterface = &deviantArt{
		Module: module,
		userGalleryPattern: regexp.MustCompile(
			`https://www\.deviantart\.com/([^/?&]+)(/gallery((/|/\?catpath=/)?))?$`,
		),
	}

	// register module to log formatter
	formatter.AddFieldMatchColorScheme("module", &formatter.FieldMatch{
		Value: module.Key(),
		Color: "232:34",
	})

	return module
}

// InitializeModule initializes the module
func (m *deviantArt) InitializeModule() {
	// set the module implementation for access to the session, database, etc
	m.deviantArtSession = session.NewSession(m.Key())
	m.Session = m.deviantArtSession

	// set the proxy if requested
	raven.CheckError(m.Session.SetProxy(m.GetProxySettings()))
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

// AddSettingsCommand adds custom module specific settings and commands to our application
func (m *deviantArt) AddSettingsCommand(command *cobra.Command) {
	m.AddProxyCommands(command)
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
	placebo, apiErr, err := m.Placebo()

	// check placebo response if the token can be used
	m.LoggedIn = apiErr == nil && err == nil && placebo.Status == "success"
	m.TriedLogin = true

	return m.LoggedIn
}

// prepareSessionForOAuth2 is used if the OAuth2 step should be completed automatically
// so we log in into the website with the session before retrieving the OAuth2 Code
// if login fails we use the browser solution as fallback which wouldn't even require a user in the database
func (m *deviantArt) prepareSessionForOAuth2(account *models.Account) bool {
	res, err := m.Session.Get("https://www.deviantart.com/users/login")
	raven.CheckError(err)

	info, err := m.getLoginCSRFToken(res)
	raven.CheckError(err)
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
	return strings.Contains(html, "\"loggedIn\":true") || strings.Contains(html, "\\\"isLoggedIn\\\":true")
}

// getLoginCSRFToken returns the CSRF token from the login site to use in our POST login request
func (m *deviantArt) getLoginCSRFToken(res *http.Response) (loginInfo loginInfo, err error) {
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
			var s string
			s, err = strconv.Unquote(jsonPattern.FindStringSubmatch(scriptContent)[1])
			if err != nil {
				return
			}
			err = json.Unmarshal([]byte(s), &loginInfo)
			if err != nil {
				return
			}
		}
	})
	return loginInfo, err
}

// Parse parses the tracked item
func (m *deviantArt) Parse(item *models.TrackedItem) (err error) {
	// special behaviour: viewing "all" gallery doesn't contain unique gallery id (using featured uuid)
	// so it won't retrieve all items, only the featured ones
	// we use the /gallery/all API endpoint in that case to retrieve actually all deviations
	if m.userGalleryPattern.MatchString(item.URI) {
		return m.parseGalleryAll(item)
	}

	appURL := item.URI
	if !strings.HasPrefix(appURL, "DeviantArt://") {
		var exists bool
		appURL, exists, err = m.getAppURL(item.URI)
		if err != nil {
			return err
		}
		if !exists {
			log.WithField("module", m.Key()).Warnf("couldn't extract app url from page %s", item.URI)
			// couldn't extract url from passed uri
			return nil
		}
	}

	switch {
	case strings.HasPrefix(appURL, "DeviantArt://collection/"):
		return m.parseCollection(appURL, item)
	case strings.HasPrefix(appURL, "DeviantArt://tag/"):
		return m.parseTag(appURL, item)
	case strings.HasPrefix(appURL, "DeviantArt://deviation/"):
		return m.parseDeviation(appURL, item)
	case strings.HasPrefix(appURL, "DeviantArt://gallery/"):
		return m.parseGallery(appURL, item)
	case strings.HasPrefix(appURL, "DeviantArt://watchfeed"):
		return m.parseFeed(item)
	}

	return nil
}

// getAppURL extracts the meta attribute da:appurl and returns it
func (m *deviantArt) getAppURL(uri string) (appURL string, exists bool, err error) {
	res, err := m.Session.Get(uri)
	if err != nil {
		return "", false, err
	}

	doc := m.Session.GetDocument(res)
	appURL, exists = doc.Find("meta[property=\"da:appurl\"]").First().Attr("content")
	return appURL, exists, nil
}
