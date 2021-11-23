// Package chounyuu contains the implementation of the chounyuu module
package chounyuu

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	formatter "github.com/DaRealFreak/colored-nested-formatter"
	"github.com/DaRealFreak/watcher-go/internal/http/session"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules"
	"github.com/DaRealFreak/watcher-go/internal/modules/chounyuu/api"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/spf13/cobra"
)

// chounyuu contains the implementation of the ModuleInterface
type chounyuu struct {
	*models.Module
	api api.ChounyuuAPI
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
			regexp.MustCompile("superfuta.com"),
		},
	}
	module.ModuleInterface = &chounyuu{
		Module: module,
		api:    api.ChounyuuAPI{},
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
	m.api.Session = m.Session

	// set the proxy if requested
	raven.CheckError(m.Session.SetProxy(m.GetProxySettings()))

	// set referer to new transport method
	client := m.Session.GetClient()
	client.Transport = m.SetReferer(client.Transport)
}

// AddModuleCommand adds custom module specific settings and commands to our application
func (m *chounyuu) AddModuleCommand(command *cobra.Command) {
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
	for _, domain := range []string{api.ChounyuuDomain, api.SuperFutaDomain} {
		// access login page for CSRF token
		_, err := m.Session.Get(fmt.Sprintf("https://g.%s/account", domain))
		if err != nil {
			m.TriedLogin = true
			return false
		}

		loginData := loginFormData{}
		loginData.IB = 1
		loginData.Name = account.Username
		loginData.Password = account.Password

		data, _ := json.Marshal(loginData)
		req, _ := http.NewRequest(
			"POST",
			fmt.Sprintf("https://g.%s/api/post/login", domain),
			bytes.NewReader(data),
		)

		req.Header.Set("Accept", "application/json, text/plain, */*")
		req.Header.Set("Content-Type", "application/json;charset=utf-8")

		client := m.Session.GetClient()
		loginUrl, _ := url.Parse(fmt.Sprintf("https://g.%s/api/post/login", domain))
		cookies := client.Jar.Cookies(loginUrl)
		for _, cookie := range cookies {
			if cookie.Name == "XSRF-TOKEN" {
				req.Header.Set("X-XSRF-TOKEN", cookie.Value)
			}
		}

		res, err := client.Do(req)
		if err != nil {
			m.TriedLogin = true
			return false
		}

		htmlResponse, _ := m.Session.GetDocument(res).Html()
		m.LoggedIn = strings.Contains(htmlResponse, "Login successful")
		m.TriedLogin = true
	}

	return m.LoggedIn
}

// Parse parses the tracked item
func (m *chounyuu) Parse(item *models.TrackedItem) error {
	if strings.Contains(item.URI, "/tag/") {
		return m.parseTag(item)
	} else if strings.Contains(item.URI, "/thread/") {
		return m.parseThread(item)
	}

	return nil
}
