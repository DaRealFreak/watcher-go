package pixiv

import (
	"encoding/json"
	"fmt"
	"github.com/DaRealFreak/watcher-go/pkg/database"
	"github.com/DaRealFreak/watcher-go/pkg/http_wrapper"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

type user struct {
	ProfileImageUrls map[string]string `json:"profile_image_urls"`
	Id               string            `json:"id"`
	Name             string            `json:"name"`
	Account          string            `json:"account"`
	MailAddress      string            `json:"mail_address"`
	IsPremium        bool              `json:"is_premium"`
	XRestrict        json.Number       `json:"x_restrict"`
	IsMailAuthorized bool              `json:"is_mail_authorized"`
}

type loginResponseData struct {
	AccessToken  string      `json:"access_token"`
	ExpiresIn    json.Number `json:"expires_in"`
	TokenType    string      `json:"token_type"`
	Scope        string      `json:"scope"`
	RefreshToken string      `json:"refresh_token"`
	User         user        `json:"user"`
	DeviceToken  string      `json:"device_token"`
}

type loginResponse struct {
	Response *loginResponseData `json:"response"`
}

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
		data.Set("username", account.Username)
		data.Set("password", account.Password)
	}
	res, err := m.post(m.mobileClient.oauthUrl, data)
	if err != nil {
		log.Fatal(err)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	var response loginResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(response)
	return false
}

func (m *pixiv) Parse(item *models.TrackedItem) {
	fmt.Println(item)
}

// custom GET function to set headers like the mobile app
func (m *pixiv) get(uri string) (*http.Response, error) {
	req, _ := http.NewRequest("GET", uri, nil)
	for headerKey, headerValue := range m.mobileClient.headers {
		req.Header.Set(headerKey, headerValue)
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
	res, err := m.Session.Client.Do(req)
	return res, err
}
