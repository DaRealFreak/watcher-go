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
	"github.com/DaRealFreak/watcher-go/pkg/http/session"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/spf13/cobra"
)

// giantessWorld contains the implementation of the ModuleInterface
type giantessWorld struct {
	models.Module
}

// NewModule generates new module and registers the URI schema
func NewModule(dbIO models.DatabaseInterface, uriSchemas map[string][]*regexp.Regexp) *models.Module {
	// register empty sub module to point to
	var subModule = giantessWorld{}

	// initialize the Module with the session/database and login status
	module := models.Module{
		DbIO:            dbIO,
		LoggedIn:        false,
		ModuleInterface: &subModule,
	}

	// set the module implementation for access to the session, database, etc
	subModule.Module = module
	gwSession := session.NewSession(subModule.GetProxySettings())
	gwSession.ModuleKey = subModule.Key()
	subModule.Session = gwSession

	// register the uri schema
	module.RegisterURISchema(uriSchemas)
	// register module to log formatter
	formatter.AddFieldMatchColorScheme("module", &formatter.FieldMatch{
		Value: module.Key(),
		Color: "232:203",
	})

	return &module
}

// Key returns the module key
func (m *giantessWorld) Key() (key string) {
	return "giantessworld.net"
}

// RequiresLogin checks if this module requires a login to work
func (m *giantessWorld) RequiresLogin() (requiresLogin bool) {
	return false
}

// IsLoggedIn returns the logged in status
func (m *giantessWorld) IsLoggedIn() bool {
	return m.LoggedIn
}

// RegisterURISchema adds our pattern to the URI Schemas
func (m *giantessWorld) RegisterURISchema(uriSchemas map[string][]*regexp.Regexp) {
	uriSchemas[m.Key()] = []*regexp.Regexp{
		regexp.MustCompile("giantessworld.net"),
	}
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
	res, _ := m.Session.GetClient().Do(req)
	htmlResponse, _ := m.Session.GetDocument(res).Html()
	m.LoggedIn = strings.Contains(htmlResponse, "Member Account")

	return m.LoggedIn
}

// Parse parses the tracked item
func (m *giantessWorld) Parse(item *models.TrackedItem) error {
	switch {
	case strings.Contains(item.URI, "viewuser.php"):
	case strings.Contains(item.URI, "browse.php"):
	case strings.Contains(item.URI, "viewstory.php"):
	}

	return nil
}
