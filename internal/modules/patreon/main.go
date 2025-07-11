// Package patreon contains the implementation of the patreon module
package patreon

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	http "github.com/bogdanfinn/fhttp"
	"net/url"
	"os"
	"regexp"
	"strings"
	"unicode"

	formatter "github.com/DaRealFreak/colored-nested-formatter"
	"github.com/DaRealFreak/watcher-go/internal/http/tls_session"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// patreon contains the implementation of the ModuleInterface
type patreon struct {
	*models.Module
	loginCsrfPattern    *regexp.Regexp
	creatorIdPattern    *regexp.Regexp
	creatorUriPattern   *regexp.Regexp
	normalizedUriRegexp *regexp.Regexp
	settings            patreonSettings
}

type patreonSettings struct {
	Cloudflare struct {
		UserAgent string `mapstructure:"user_agent"`
	} `mapstructure:"cloudflare"`
	ConvertNameToId bool `mapstructure:"convert_name_to_id"`
	ExternalURLs    struct {
		DownloadExternalItems     bool `mapstructure:"download_external_items"`
		PrintExternalItems        bool `mapstructure:"print_external_items"`
		SkipErrorsForExternalURLs bool `mapstructure:"skip_errors_for_external_urls"`
	} `mapstructure:"external_urls"`
}

type loginAttributes struct {
	AuthContext string `json:"auth_context"`
	PatreonAuth struct {
		AllowAccountCreation bool   `json:"allow_account_creation"`
		Email                string `json:"email"`
		Password             string `json:"password"`
	} `json:"patreon_auth"`
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
			regexp.MustCompile(`https://www.patreon.com`),
			regexp.MustCompile(`(https://www.|^)patreon.com`),
			regexp.MustCompile("patreon://creator/"),
		},
	}
	module.ModuleInterface = &patreon{
		Module:              module,
		loginCsrfPattern:    regexp.MustCompile(`"csrfSignature":"(.*?)","`),
		creatorIdPattern:    regexp.MustCompile(`"creator":\{"data":\{"id":"(?P<ID>\d+)"`),
		creatorUriPattern:   regexp.MustCompile(`patreon://creator/(\d+)`),
		normalizedUriRegexp: regexp.MustCompile(`patreon://creator/(\d+)/.*`),
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
	// initialize settings
	raven.CheckError(viper.UnmarshalKey(
		fmt.Sprintf("Modules.%s", m.GetViperModuleKey()),
		&m.settings,
	))

	// already initialized
	if m.Session != nil {
		return
	}

	// initialize session
	m.Session = tls_session.NewTlsClientSession(m.Key)

	// set the proxy if requested
	raven.CheckError(m.Session.SetProxy(m.GetProxySettings()))
}

func (m *patreon) AddItem(uri string) (string, error) {
	// we require the session to check for the creator ID
	if m.Session == nil {
		if err := m.Load(); err != nil {
			return "", err
		}

		m.InitializeModule()

		// we're not converting the uri
		if !m.settings.ConvertNameToId {
			return uri, nil
		}

		if !m.LoggedIn {
			// login if we have an account, we can't extract the creator ID without an account
			account := m.DbIO.GetAccount(m)
			if account == nil {
				return uri, fmt.Errorf("an account is required for the extraction of the creator ID")
			}

			m.Login(account)
		}
	}

	if !m.normalizedUriRegexp.MatchString(uri) {
		creatorId, idErr := m.getCreatorID(uri)
		if idErr != nil {
			return uri, idErr
		}

		creatorName, nameErr := m.getCreatorName(uri)
		if nameErr != nil {
			return uri, nameErr
		}

		return fmt.Sprintf("patreon://creator/%d/%s", creatorId, creatorName), nil
	} else {
		return uri, nil
	}
}

// AddModuleCommand adds custom module specific settings and commands to our application
func (m *patreon) AddModuleCommand(command *cobra.Command) {
	m.AddProxyCommands(command)
}

// Login logs us in for the current session if possible/account available
func (m *patreon) Login(account *models.Account) bool {
	// check if we have a cookie of the name session_id for the domain patreon.com
	mainUrl, _ := url.Parse("https://www.patreon.com")
	foundCookie := false
	for _, cookie := range m.Session.GetCookies(mainUrl) {
		if cookie.Name == "session_id" {
			foundCookie = true
			break
		}
	}

	if foundCookie {
		log.WithField("module", m.Key).Info("using existing session cookie")
		m.LoggedIn = true
		return m.LoggedIn
	}

	formData := loginFormData{
		Data: loginData{
			Type: "genericPatreonApi",
			Attributes: loginAttributes{
				AuthContext: "auth",
				PatreonAuth: struct {
					AllowAccountCreation bool   `json:"allow_account_creation"`
					Email                string `json:"email"`
					Password             string `json:"password"`
				}{
					AllowAccountCreation: false,
					Email:                account.Username,
					Password:             account.Password,
				},
			},
			Relationships: struct{}{},
		},
	}

	data, err := json.Marshal(formData)
	if err != nil {
		log.WithField("module", m.Key).Error(err)
	}

	res, err := m.get("https://www.patreon.com/login")
	if err != nil {
		log.WithField("module", m.Key).Error(err)
		return false
	}

	loginCsrfMatches := m.loginCsrfPattern.FindStringSubmatch(m.Session.GetDocument(res).Text())
	if len(loginCsrfMatches) != 2 {
		log.WithField("module", m.Key).Fatal(
			fmt.Errorf("unexpected amount of matches in search of login CSRF token"),
		)
		return false
	}

	req, _ := http.NewRequest(
		"POST",
		"https://www.patreon.com/api/auth?include=user.null&fields[user]=[]&json-api-version=1.0&json-api-use-default-includes=false",
		bytes.NewReader(data),
	)
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
				verificationRes, verificationError := m.get(strings.TrimSuffix(text, "\n"))
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

	if err = json.Unmarshal([]byte(loginRes), &loginSuccess); err != nil {
		log.WithField("module", m.Key).Error(err)
	}

	m.LoggedIn = loginSuccess.Data.ID.String() != ""

	return m.LoggedIn
}

// Parse parses the tracked item
func (m *patreon) Parse(item *models.TrackedItem) error {
	if m.settings.ConvertNameToId && !m.normalizedUriRegexp.MatchString(item.URI) {
		newUri, err := m.AddItem(item.URI)
		if err == nil {
			m.DbIO.ChangeTrackedItemUri(item, newUri)
		} else {
			log.WithField("module", item.Module).Warningf(
				"unable to convert campaign URL to ID for %s (%s)", item.URI, err.Error(),
			)
		}
	}

	return m.parseCampaign(item)
}
