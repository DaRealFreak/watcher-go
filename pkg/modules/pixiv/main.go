package pixiv

import (
	"encoding/json"
	"fmt"
	"github.com/DaRealFreak/watcher-go/pkg/database"
	"github.com/DaRealFreak/watcher-go/pkg/http_wrapper"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

type mobileClient struct {
	oauthUrl     string
	headers      map[string]string
	clientId     string
	clientSecret string
	accessToken  string
	refreshToken string
}

type pixiv struct {
	models.Module
	mobileClient *mobileClient
}

// generate new module and register uri schema
func NewModule(dbIO *database.DbIO, uriSchemas map[string][]*regexp.Regexp) *models.Module {
	// register empty sub module to point to
	var subModule = pixiv{
		mobileClient: &mobileClient{
			oauthUrl: "https://oauth.secure.pixiv.net/auth/token",
			headers: map[string]string{
				"App-OS":         "ios",
				"App-OS-Version": "10.3.1",
				"App-Version":    "6.7.1",
				"User-Agent":     "PixivIOSApp/6.7.1 (iOS 10.3.1; iPhone8,1)",
				"Referer":        "https://app-api.pixiv.net/",
				"Content-Type":   "application/x-www-form-urlencoded",
			},
			clientId:     "MOBrBDS8blbauoSck0ZfDbtuzpyT",
			clientSecret: "lsACyCD94FhDUtGTXi3QzcFE2uU1hqtDaKeqrdwj",
			accessToken:  "",
			refreshToken: "",
		},
	}

	// initialize the Module with the session/database and login status
	module := models.Module{
		DbIO:            dbIO,
		Session:         http_wrapper.NewSession(),
		LoggedIn:        false,
		ModuleInterface: &subModule,
	}
	// set the module implementation for access to the session, database, etc
	subModule.Module = module
	// register the uri schema
	module.RegisterUriSchema(uriSchemas)
	return &module
}

// retrieve the module key
func (m *pixiv) Key() (key string) {
	return "pixiv.net"
}

// check if this module requires a login to work
func (m *pixiv) RequiresLogin() (requiresLogin bool) {
	return true
}

// retrieve the logged in status
func (m *pixiv) IsLoggedIn() (LoggedIn bool) {
	return m.LoggedIn
}

// add our pattern to the uri schemas
func (m *pixiv) RegisterUriSchema(uriSchemas map[string][]*regexp.Regexp) {
	var moduleUriSchemas []*regexp.Regexp
	test, _ := regexp.Compile(".*pixiv.(co.jp|net)")
	moduleUriSchemas = append(moduleUriSchemas, test)
	uriSchemas[m.Key()] = moduleUriSchemas
}

// login function
func (m *pixiv) Login(account *models.Account) bool {
	data := url.Values{
		"get_secure_url": {"1"},
		"client_id":      {m.mobileClient.clientId},
		"client_secret":  {m.mobileClient.clientSecret},
	}

	if m.mobileClient.refreshToken != "" {
		data.Set("grant_type", "refresh_token")
		data.Set("refresh_token", m.mobileClient.refreshToken)
	} else {
		data.Set("grant_type", "password")
		data.Set("username", account.Username+"nope")
		data.Set("password", account.Password)
	}

	res, err := m.post(m.mobileClient.oauthUrl, data)
	m.CheckError(err)

	body, err := ioutil.ReadAll(res.Body)
	m.CheckError(err)

	var response loginResponse
	_ = json.Unmarshal(body, &response)

	// check if the response could be parsed properly and save tokens
	if response.Response != nil {
		m.LoggedIn = true
		m.mobileClient.refreshToken = response.Response.RefreshToken
		m.mobileClient.accessToken = response.Response.AccessToken
	} else {
		var response errorResponse
		_ = json.Unmarshal(body, &response)
		log.Warning("login not successful.")
		log.Fatalf("message: %s (code: %s)",
			response.Errors.System.Message,
			response.Errors.System.Code.String(),
		)
	}
	return m.LoggedIn
}

func (m *pixiv) Parse(item *models.TrackedItem) {
	fmt.Println(item)
}

// custom GET function to set headers like the mobile app
func (m *pixiv) get(uri string) (*http.Response, error) {
	req, _ := http.NewRequest("GET", uri, nil)
	for headerKey, headerValue := range m.mobileClient.headers {
		req.Header.Add(headerKey, headerValue)
	}
	if m.mobileClient.accessToken != "" {
		req.Header.Add("Authorization", "Bearer "+m.mobileClient.accessToken)
	}
	res, err := m.Session.Client.Do(req)
	return res, err
}

// custom GET function to set headers like the mobile app
func (m *pixiv) post(uri string, data url.Values) (*http.Response, error) {
	req, _ := http.NewRequest("POST", uri, strings.NewReader(data.Encode()))
	for headerKey, headerValue := range m.mobileClient.headers {
		req.Header.Add(headerKey, headerValue)
	}
	if m.mobileClient.accessToken != "" {
		req.Header.Add("Authorization", "Bearer "+m.mobileClient.accessToken)
	}
	res, err := m.Session.Client.Do(req)
	return res, err
}
