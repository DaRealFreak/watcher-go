// Package giantessworld contains the implementation of the giantessworld module
package giantessworld

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	formatter "github.com/DaRealFreak/colored-nested-formatter"
	"github.com/DaRealFreak/watcher-go/internal/http/session"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/spf13/cobra"
)

// giantessWorld contains the implementation of the ModuleInterface
type giantessWorld struct {
	*models.Module
	baseURL              *url.URL
	chapterUpdatePattern *regexp.Regexp
}

// nolint: gochecknoinits
// init function registers the bare and the normal module to the module factories
func init() {
	modules.GetModuleFactory().RegisterModule(NewBareModule())
}

// NewBareModule returns a bare module implementation for the CLI options
func NewBareModule() *models.Module {
	module := &models.Module{
		Key:           "giantessworld.net",
		RequiresLogin: false,
		LoggedIn:      false,
		URISchemas: []*regexp.Regexp{
			regexp.MustCompile("giantessworld.net"),
		},
	}
	module.ModuleInterface = &giantessWorld{
		Module:               module,
		chapterUpdatePattern: regexp.MustCompile(`Updated:\s+(\w+ \d{2} \d{4})+`),
	}

	// register module to log formatter
	formatter.AddFieldMatchColorScheme("module", &formatter.FieldMatch{
		Value: module.Key,
		Color: "232:203",
	})

	return module
}

// InitializeModule initializes the module
func (m *giantessWorld) InitializeModule() {
	// set the module implementation for access to the session, database, etc
	m.baseURL, _ = url.Parse("https://www.giantessworld.net")
	m.Session = session.NewSession(m.Key)

	// set the proxy if requested
	raven.CheckError(m.Session.SetProxy(m.GetProxySettings()))
}

// AddSettingsCommand adds custom module specific settings and commands to our application
func (m *giantessWorld) AddSettingsCommand(command *cobra.Command) {
	m.AddProxyCommands(command)
}

// Login logs us in for the current session if possible/account available
func (m *giantessWorld) Login(account *models.Account) bool {
	values := url.Values{
		"penname":  {account.Username},
		"password": {account.Password},
		"submit":   {"Go"},
	}

	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	for formName, formValue := range values {
		w, err := w.CreateFormField(formName)
		if err != nil {
			return false
		}

		_, err = w.Write([]byte(formValue[0]))
		if err != nil {
			return false
		}
	}

	req, _ := http.NewRequest("POST", "http://www.giantessworld.net/user.php?action=login", &b)

	req.Header.Add("Content-Type", w.FormDataContentType())

	res, err := m.Session.GetClient().Do(req)
	raven.CheckError(err)

	htmlResponse, _ := m.Session.GetDocument(res).Html()
	m.LoggedIn = strings.Contains(htmlResponse, "Member Account")
	m.TriedLogin = true

	return m.LoggedIn
}

// Parse parses the tracked item
func (m *giantessWorld) Parse(item *models.TrackedItem) error {
	switch {
	case strings.Contains(item.URI, "viewuser.php"), strings.Contains(item.URI, "browse.php"):
		return m.parseUser(item)
	case strings.Contains(item.URI, "viewstory.php"):
		return m.parseStory(item)
	}

	return nil
}
