// Package patreon contains the implementation of the patreon module
package patreon

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"

	cloudflarebp "github.com/DaRealFreak/cloudflare-bp-go"
	formatter "github.com/DaRealFreak/colored-nested-formatter"
	"github.com/DaRealFreak/watcher-go/pkg/http/session"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/DaRealFreak/watcher-go/pkg/modules"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
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

type loginErrorResponse struct {
	Errors []*struct {
		Code     json.Number `json:"code"`
		CodeName string      `json:"code_name"`
		Detail   string      `json:"detail"`
	} `json:"errors"`
}

type loginSuccessResponse struct {
	Data struct {
		ID json.Number `json:"id"`
	} `json:"data"`
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

	// add CloudFlare bypass
	m.Session.GetClient().Transport = cloudflarebp.AddCloudFlareByPass(m.Session.GetClient().Transport)
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
		"https://www.patreon.com/api/login?include=campaign%2Cuser_location&json-api-version=1.0",
		"application/vnd.api+json",
		bytes.NewReader(data),
	)
	if err != nil {
		log.WithField("module", m.Key).Error(err)
	}

	loginRes := m.Session.GetDocument(res).Text()

	var (
		loginError   loginErrorResponse
		loginSuccess loginSuccessResponse
	)

	if err := json.Unmarshal([]byte(loginRes), &loginError); err != nil {
		log.WithField("module", m.Key).Error(err)
	}

	if len(loginError.Errors) > 0 {
		for _, err := range loginError.Errors {
			log.WithField("module", m.Key).Fatal(
				fmt.Errorf("error occurred during login (code: %s): %s", err.Code, err.Detail),
			)
		}
	}

	if err := json.Unmarshal([]byte(loginRes), &loginSuccess); err != nil {
		log.WithField("module", m.Key).Error(err)
	}

	m.LoggedIn = loginSuccess.Data.ID.String() != ""

	return m.LoggedIn
}

// Parse parses the tracked item
func (m *patreon) Parse(item *models.TrackedItem) error {
	return m.parseCampaign(item)
}
