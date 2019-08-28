package deviantart

import (
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/DaRealFreak/watcher-go/pkg/raven"
	"github.com/DaRealFreak/watcher-go/pkg/webserver"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

// WebServerTimeout default timeout for web server to retrieve OAuth2 token
const WebServerTimeout = 60 * time.Second

// APIClientID is the client ID of the registered application accessing the API
// can be retrieved from https://www.deviantart.com/developers/apps
// make sure the OAuth2 Grant Type is set to Implicit and the redirect URL matches the constant below
const APIClientID = "9991"

// RedirectURL is the URL to the local DNS resolved page (lvh is resolving to 127.0.0.1, only works with IPv4 though)
// Implicit grant type requires https URLs and has to match the pattern exactly contrary to the authorization code
const RedirectURL = "https://lvh.me:8080/da-cb"

// oAuth2Check is used to retrieve the OAuth2 code
type oAuth2Check struct {
	code    *oauth2.Token
	granted chan bool
}

// newOAuth2Application creates the granted channel
func newOAuth2Application() *oAuth2Check {
	return &oAuth2Check{
		granted: make(chan bool),
	}
}

// checkResponseUrlForTokenFragment checks the passed http Response for OAuth2 Token fragments
func (a *oAuth2Check) checkResponseUrlForTokenFragment(res *http.Response) (foundToken bool) {
	f, _ := url.ParseQuery(res.Request.URL.Fragment)
	if len(f["access_token"]) > 0 && len(f["token_type"][0]) > 0 && len(f["expires_in"][0]) > 0 {
		expiration, err := strconv.Atoi(f["expires_in"][0])
		raven.CheckError(err)

		a.code = &oauth2.Token{
			AccessToken:  f["access_token"][0],
			TokenType:    f["token_type"][0],
			RefreshToken: "",
			Expiry:       time.Now().Add(time.Duration(expiration) * time.Second),
		}
		a.granted <- true
		return true
	}
	return false
}

// oAuth2ApplicationCallback handles web requests
// not much to do here since the client has to check for the OAuth2 token fragments
func (a *oAuth2Check) oAuth2ApplicationCallback(w http.ResponseWriter, r *http.Request) {
	log.Debug("request received from: ", r.RemoteAddr)
}

// retrieveOAuth2Code creates a new OAuth2Application, starts a local web server on port 8080
// and waits for a request containing the OAuth2 token for the API
// server will shut down after 60 seconds automatically and return an empty string
func (m *deviantArt) retrieveOAuth2Code() *oauth2.Token {
	oAuth2Application := newOAuth2Application()
	webserver.Server.Mux.HandleFunc("/da-cb", oAuth2Application.oAuth2ApplicationCallback)
	webserver.StartWebServer()

	oAuth2URL := "https://www.deviantart.com/oauth2/authorize?response_type=token" +
		"&client_id=" + APIClientID + "&redirect_uri=" + RedirectURL + "&scope=basic&state=mysessionid"
	// send request and wait for either a successful response or a timeout
	go m.sendOAuth2AcceptRequest(oAuth2Application, oAuth2URL)

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
func (m *deviantArt) sendOAuth2AcceptRequest(a *oAuth2Check, oAuth2URL string) {
	// open the passed OAuth2 URL
	res, err := m.Session.Get(oAuth2URL)
	raven.CheckError(err)

	// if application already got authorized we are getting redirected directly to the OAuth2 token URL
	if a.checkResponseUrlForTokenFragment(res) {
		log.Debugf("retrieved OAuth2 token: %s", a.code.AccessToken)
		return
	}

	// we are currently in the authorization step
	// if the function didn't get redirected directly to the OAuth2 token URL
	log.Info("application not authorized yet, sending authorization POST request")
	doc := m.Session.GetDocument(res)
	form := doc.Find("form#authorize_form").First()

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
	res, err = m.Session.Post("https://www.deviantart.com/settings/authorize_app", values)
	raven.CheckError(err)

	// check the response to the authorization request for the OAuth2 token URL
	a.checkResponseUrlForTokenFragment(res)
}
