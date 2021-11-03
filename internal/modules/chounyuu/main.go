// Package chounyuu contains the implementation of the chounyuu module
package chounyuu

import (
	"bytes"
	"encoding/json"
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

// chounyuu contains the implementation of the ModuleInterface
type chounyuu struct {
	*models.Module
}

// nolint: gochecknoinits
// init function registers the bare and the normal module to the module factories
func init() {
	modules.GetModuleFactory().RegisterModule(NewBareModule())
}

// NewBareModule returns a bare module implementation for the CLI options
func NewBareModule() *models.Module {
	module := &models.Module{
		Key:           "chounyuu.com",
		RequiresLogin: false,
		LoggedIn:      false,
		URISchemas: []*regexp.Regexp{
			regexp.MustCompile("chounyuu.com"),
		},
	}
	module.ModuleInterface = &chounyuu{
		Module: module,
	}

	// register module to log formatter
	formatter.AddFieldMatchColorScheme("module", &formatter.FieldMatch{
		Value: module.Key,
		Color: "0:218",
	})

	return module
}

// InitializeModule initializes the module
func (m *chounyuu) InitializeModule() {
	// set the module implementation for access to the session, database, etc
	m.Session = session.NewSession(m.Key)

	// set the proxy if requested
	raven.CheckError(m.Session.SetProxy(m.GetProxySettings()))

	// set referer to new transport method
	client := m.Session.GetClient()
	client.Transport = m.SetReferer(client.Transport)
}

// AddSettingsCommand adds custom module specific settings and commands to our application
func (m *chounyuu) AddSettingsCommand(command *cobra.Command) {
	m.AddProxyCommands(command)
}

// loginData is the json struct for the JSON login form
type loginFormData struct {
	IB       int    `json:"ib"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

// Login logs us in for the current session if possible/account available
func (m *chounyuu) Login(account *models.Account) bool {
	// access login page for CSRF token
	res, err := m.Session.Get("https://g.chounyuu.com/account")
	if err != nil {
		m.TriedLogin = true
		return false
	}

	loginData := loginFormData{}
	loginData.IB = 1
	loginData.Name = account.Username
	loginData.Password = account.Password

	data, _ := json.Marshal(loginData)
	req, _ := http.NewRequest("POST", "https://g.chounyuu.com/api/post/login", bytes.NewReader(data))

	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Content-Type", "application/json;charset=utf-8")

	client := m.Session.GetClient()
	loginUrl, _ := url.Parse("https://g.chounyuu.com/api/post/login")
	cookies := client.Jar.Cookies(loginUrl)
	for _, cookie := range cookies {
		if cookie.Name == "XSRF-TOKEN" {
			req.Header.Set("X-XSRF-TOKEN", cookie.Value)
		}
	}

	res, err = client.Do(req)
	if err != nil {
		m.TriedLogin = true
		return false
	}

	htmlResponse, _ := m.Session.GetDocument(res).Html()
	m.LoggedIn = strings.Contains(htmlResponse, "Login successful")
	m.TriedLogin = true

	return m.LoggedIn
}

// Parse parses the tracked item
func (m *chounyuu) Parse(item *models.TrackedItem) error {

	return nil
}
