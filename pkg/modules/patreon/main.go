// Package patreon contains the implementation of the patreon module
package patreon

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	formatter "github.com/DaRealFreak/colored-nested-formatter"
	"github.com/DaRealFreak/watcher-go/pkg/http/session"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/DaRealFreak/watcher-go/pkg/modules"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
	browser "github.com/EDDYCJY/fake-useragent"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// patreon contains the implementation of the ModuleInterface
type patreon struct {
	*models.Module
	creatorIPattern *regexp.Regexp
}

type loginAttributes struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginData struct {
	Relationships struct {
	} `json:"relationships"`
	Type       string          `json:"type"`
	Attributes loginAttributes `json:"attributes"`
}

// loginData is the json struct for the JSON login form
type loginFormData struct {
	Data loginData `json:"data"`
}

// nolint: gochecknoinits
// init function registers the bare and the normal module to the module factories
func init() {
	modules.GetModuleFactory().RegisterModule(NewBareModule())
}

// NewBareModule returns a bare module implementation for the CLI options
func NewBareModule() *models.Module {
	module := &models.Module{
		Key:           "patreon.com",
		RequiresLogin: false,
		LoggedIn:      false,
		URISchemas: []*regexp.Regexp{
			regexp.MustCompile(".*patreon.com"),
		},
	}
	module.ModuleInterface = &patreon{
		Module:          module,
		creatorIPattern: regexp.MustCompile(`"creator_id":\s(?P<ID>\d+)?`),
	}

	// register module to log formatter
	formatter.AddFieldMatchColorScheme("module", &formatter.FieldMatch{
		Value: module.Key,
		Color: "232:167",
	})

	return module
}

// InitializeModule initializes the module
func (m *patreon) InitializeModule() {
	// initialize session
	m.Session = session.NewSession(m.Key)

	// set the proxy if requested
	raven.CheckError(m.Session.SetProxy(m.GetProxySettings()))

	// set TLS configuration for transport layer of client to pass CloudFlare checks
	if trans, ok := m.Session.GetClient().Transport.(*http.Transport); ok {
		trans.TLSClientConfig = &tls.Config{
			PreferServerCipherSuites: true,
		}
	}

	client := m.Session.GetClient()
	client.Transport = m.SetUserAgent(client.Transport, browser.Firefox())
	client.Transport = m.SetCloudFlareHeaders(client.Transport)
}

// AddSettingsCommand adds custom module specific settings and commands to our application
func (m *patreon) AddSettingsCommand(command *cobra.Command) {
	m.AddProxyCommands(command)
}

// Login logs us in for the current session if possible/account available
func (m *patreon) Login(account *models.Account) bool {
	loginData := loginFormData{
		Data: loginData{
			Type: "user",
			Attributes: loginAttributes{
				Email:    account.Username,
				Password: account.Password,
			},
			Relationships: struct{}{},
		},
	}

	data, err := json.Marshal(loginData)
	if err != nil {
		log.WithField("module", m.Key).Error(err)
	}

	res, err := m.Session.GetClient().Post(
		"https://www.patreon.com/api/login?include=campaign,user_location&json-api-version=1.0",
		"application/vnd.api+json",
		bytes.NewReader(data),
	)
	if err != nil {
		log.WithField("module", m.Key).Error(err)
	}

	m.LoggedIn = strings.Contains(
		m.Session.GetDocument(res).Text(),
		fmt.Sprintf(`"email":"%s"`, account.Username),
	)

	return m.LoggedIn
}

// Parse parses the tracked item
func (m *patreon) Parse(item *models.TrackedItem) error {
	return m.parseCampaign(item)
}
