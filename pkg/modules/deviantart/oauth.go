package deviantart

import (
	"net/http"
	"net/url"
	"time"

	"github.com/DaRealFreak/watcher-go/pkg/raven"
	"github.com/DaRealFreak/watcher-go/pkg/webserver"
	"github.com/pkg/browser"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// WebServerTimeout default timeout for web server to retrieve OAuth2 token
const WebServerTimeout = 60 * time.Second

// APIClientID is the client ID of the registered application accessing the API
// can be retrieved from https://www.deviantart.com/developers/apps
const APIClientID = "9991"

// RedirectURL is the URL to the local DNS resolved page (lvh is resolving to 127.0.0.1, only works with IPv4 though)
const RedirectURL = "http://lvh.me:8080/da-cb"

// oAuth2 is used to retrieve the OAuth2 code
type oAuth2 struct {
	code    string
	granted chan bool
}

// newOAuth2Application creates the granted channel
func newOAuth2Application() *oAuth2 {
	return &oAuth2{
		granted: make(chan bool),
	}
}

// oAuth2ApplicationCallback handles web requests and passes it to the granted channel
// if a granted query fragment is present
func (a *oAuth2) oAuth2ApplicationCallback(w http.ResponseWriter, r *http.Request) {
	q, _ := url.ParseQuery(r.URL.RawQuery)
	if len(q["code"]) > 0 {
		a.code = q["code"][0]
		a.granted <- true
	}
	log.Debug("request received from: ", r.RemoteAddr)
}

// retrieveOAuth2Code creates a new OAuth2Application, starts a local web server on port 8080
// and waits for a request containing the OAuth2 token for the API
// server will shut down after 60 seconds automatically and return an empty string
func (m *deviantArt) retrieveOAuth2Code() string {
	oAuth2Application := newOAuth2Application()
	webserver.Server.Mux.HandleFunc("/da-cb", oAuth2Application.oAuth2ApplicationCallback)
	webserver.StartWebServer()

	oAuth2URL := "https://www.deviantart.com/oauth2/authorize?response_type=code" +
		"&client_id=" + APIClientID + "&redirect_uri=" + RedirectURL + "&scope=basic&state=mysessionid"

	switch {
	case viper.GetBool("Modules.DeviantArt.OAuth2.Auto"):
		go m.sendOAuth2AcceptRequest(oAuth2URL)
	case viper.GetBool("Modules.DeviantArt.OAuth2.Browser"):
		raven.CheckError(browser.OpenURL(oAuth2URL))
	default:
		log.Infof("open following url to retrieve the token: %s", oAuth2URL)
	}

	select {
	case <-oAuth2Application.granted:
		log.Info("callback with granted received")
	case <-time.After(WebServerTimeout):
		log.Warningf("no callback with token received within %d seconds",
			int(WebServerTimeout.Seconds()),
		)
	}
	webserver.StopWebServer()
	return oAuth2Application.code
}

// sendOAuth2AcceptRequest automates the authentication process and logs the user in
// before checking and if needed authorizing the application
// this is the default process since it requires the least amount of input from the user
func (m *deviantArt) sendOAuth2AcceptRequest(oAuth2URL string) {
	// open the passed OAuth2 URL
	res, err := m.Session.Get(oAuth2URL)
	raven.CheckError(err)

	// if already authorized we will receive a nil error and an empty response
	doc := m.Session.GetDocument(res)
	form := doc.Find("form#authorize_form").First()
	if html, err := form.Html(); err == nil && html == "" {
		log.Debug("application already authorized, skipping authorization POST request")
		return
	}

	// scrape all relevant data from the selected form
	authValues := new(authInfo)
	authValues.State, _ = form.Find("input[name=\"state\"]").First().Attr("value")
	authValues.Scope, _ = form.Find("input[name=\"scope\"]").First().Attr("value")
	authValues.ResponseType, _ = form.Find("input[name=\"response_type\"]").First().Attr("value")
	authValues.ValidateToken, _ = form.Find("input[name=\"validate_token\"]").First().Attr("value")
	authValues.ValidateKey, _ = form.Find("input[name=\"validate_key\"]").First().Attr("value")

	// pack it into values and send the post request
	values := url.Values{
		"terms_agree[]":  {"1", "0"},
		"authorized":     {"1"},
		"state":          {authValues.State},
		"scope":          {authValues.Scope},
		"redirect_uri":   {RedirectURL},
		"response_type":  {authValues.ResponseType},
		"client_id":      {APIClientID},
		"validate_token": {authValues.ValidateToken},
		"validate_key":   {authValues.ValidateKey},
	}
	_, _ = m.Session.Post("https://www.deviantart.com/settings/authorize_app", values)
}
