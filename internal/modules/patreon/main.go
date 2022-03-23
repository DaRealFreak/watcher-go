// Package patreon contains the implementation of the patreon module
package patreon

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	cloudflarebp "github.com/DaRealFreak/cloudflare-bp-go"
	formatter "github.com/DaRealFreak/colored-nested-formatter"
	"github.com/DaRealFreak/watcher-go/internal/http/session"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// patreon contains the implementation of the ModuleInterface
type patreon struct {
	*models.Module
	loginCsrfPattern  *regexp.Regexp
	creatorIdPattern  *regexp.Regexp
	creatorUriPattern *regexp.Regexp
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
			regexp.MustCompile("patreon://creator/"),
		},
	}
	module.ModuleInterface = &patreon{
		Module:            module,
		loginCsrfPattern:  regexp.MustCompile(`window\.patreon\.csrfSignature = "(.*)";`),
		creatorIdPattern:  regexp.MustCompile(`"creator_id":\s(?P<ID>\d+)?`),
		creatorUriPattern: regexp.MustCompile(`patreon://creator/([\d]+)`),
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
	cloudflareOptions := cloudflarebp.GetDefaultOptions()
	cloudflareOptions.Headers["Accept-Encoding"] = "gzip, deflate, br"
	m.Session.GetClient().Transport = cloudflarebp.AddCloudFlareByPass(m.Session.GetClient().Transport, cloudflareOptions)
}

func (m *patreon) AddItem(uri string) (string, error) {
	res, err := m.Session.Get(uri)
	if err != nil {
		return uri, err
	}

	creatorIDMatches := m.creatorIdPattern.FindStringSubmatch(m.Session.GetDocument(res).Text())
	if len(creatorIDMatches) != 2 {
		return uri, fmt.Errorf("unexpected amount of matches in search of creator id ")
	}

	creatorID, _ := strconv.ParseInt(creatorIDMatches[1], 10, 64)

	return fmt.Sprintf("patreon://creator/%d", creatorID), nil
}

// AddModuleCommand adds custom module specific settings and commands to our application
func (m *patreon) AddModuleCommand(command *cobra.Command) {
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

	res, err := m.Session.Get("https://www.patreon.com/login")
	loginCsrfMatches := m.loginCsrfPattern.FindStringSubmatch(m.Session.GetDocument(res).Text())
	if len(loginCsrfMatches) != 2 {
		log.WithField("module", m.Key).Fatal(
			fmt.Errorf("unexpected amount of matches in search of login CSRF token"),
		)
		return false
	}

	req, _ := http.NewRequest("POST", "https://www.patreon.com/api/login?include=campaign%2Cuser_location&json-api-version=1.0", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/vnd.api+json")
	req.Header.Set("X-CSRF-Signature", loginCsrfMatches[1])

	res, err = m.Session.GetClient().Do(req)
	if err != nil {
		log.WithField("module", m.Key).Error(err)
	}
	if res.StatusCode != 200 {
		log.WithField("module", m.Key).Error("unable to login to patreon.com")
	}

	loginRes := m.Session.GetDocument(res).Text()

	var (
		loginError   loginErrorResponse
		loginSuccess loginSuccessResponse
	)

	if err = json.Unmarshal([]byte(loginRes), &loginError); err != nil {
		log.WithField("module", m.Key).Error(err)
	}

	if len(loginError.Errors) > 0 {
		for _, loginErr := range loginError.Errors {
			if loginErr.Code.String() == "111" {
				reader := bufio.NewReader(os.Stdin)
				fmt.Print("Please enter the verification link from the e-mail: ")
				text, _ := reader.ReadString('\n')
				// remove control characters
				text = strings.TrimFunc(text, func(r rune) bool {
					return !unicode.IsGraphic(r)
				})
				verificationRes, verificationError := m.Session.Get(strings.TrimSuffix(text, "\n"))
				if verificationError != nil {
					log.WithField("module", m.Key).Fatal(
						fmt.Errorf("error occurred during login (code: %s): %s", loginErr.Code, loginErr.Detail),
					)
					return false
				}

				if verificationRes.StatusCode == 200 {
					return m.Login(account)
				}
			}

			log.WithField("module", m.Key).Fatal(
				fmt.Errorf("error occurred during login (code: %s): %s", loginErr.Code, loginErr.Detail),
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
