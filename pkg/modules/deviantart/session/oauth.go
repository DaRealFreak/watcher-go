package session

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/DaRealFreak/watcher-go/pkg/raven"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

// Timeout is default timeout duration to check redirects of the client for OAuth2 token fragments for
const Timeout = 60 * time.Second

// APIClientID is the client ID of the registered application accessing the API
// can be retrieved from https://www.deviantart.com/developers/apps
// make sure the OAuth2 Grant Type is set to Implicit and the redirect URL matches the constant below
const APIClientID = "9991"

// RedirectURL is the URL the API tries to redirect you to. Doesn't have to exist, we simply check the redirects.
// Implicit grant type requires https URLs and has to match the pattern exactly contrary to the authorization code
// in case you want to register your own application on paranoia
const RedirectURL = "https://lvh.me:8080/da-cb"

// tokenRequestApplication is used to retrieve the OAuth2 code
type tokenRequestApplication struct {
	token           *oauth2.Token
	granted         chan bool
	sessionRedirect func(req *http.Request, via []*http.Request) error
}

// authInfo contains all required information for the app authorization request
type authInfo struct {
	State         string
	Scope         string
	ResponseType  string
	ValidateToken string
	ValidateKey   string
}

// newTokenRequestApplication creates the granted channel and functions for the whole OAuth2 token check process
func newTokenRequestApplication() *tokenRequestApplication {
	return &tokenRequestApplication{
		granted: make(chan bool),
	}
}

// checkRequestForTokenFragment checks the passed http Request for OAuth2 Token fragments
// if not found it uses the redirect default behaviour
func (a *tokenRequestApplication) checkRequestForTokenFragment(res *http.Request) (foundToken bool) {
	f, _ := url.ParseQuery(res.URL.Fragment)
	if len(f["access_token"]) > 0 && len(f["token_type"][0]) > 0 && len(f["expires_in"][0]) > 0 {
		expiration, err := strconv.Atoi(f["expires_in"][0])
		raven.CheckError(err)

		a.token = &oauth2.Token{
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

// checkRedirect checks the passed request for token fragments and returns http.ErrUseLastResponse if found
// this causes the client to not follow the redirect, enabling us to use non-existing URLs as redirect URL
func (a *tokenRequestApplication) checkRedirect(req *http.Request, via []*http.Request) error {
	if a.checkRequestForTokenFragment(req) {
		log.Debugf("found OAuth2 token in redirect: %s, "+
			"preventing redirect to avert error messages from unresolved/unreachable domain", a.token.AccessToken)
		return http.ErrUseLastResponse
	}

	// return the previously set redirect function if no token fragments were in the request
	if a.sessionRedirect != nil {
		return a.sessionRedirect(req, via)
	}
	// session redirect can be nil too, fallback to default http.Client -> defaultCheckRedirect
	if len(via) >= 10 {
		return errors.New("stopped after 10 redirects")
	}
	return nil
}

// retrieveOAuth2Token creates a new OAuth2Application and checks redirects for OAuth2 token fragments
// if not found it tries to authorize the application and checks again
// if token could not get extracted within 60 seconds it will return a nil token
func (s *DeviantArtSession) retrieveOAuth2Token(scope string) *oauth2.Token {
	tokenRequestApplication := newTokenRequestApplication()
	oAuth2URL := fmt.Sprintf("https://www.deviantart.com/oauth2/authorize"+
		"?response_type=token&client_id=%s&redirect_uri=%s&scope=%s&state=mysessionid", APIClientID, RedirectURL, url.QueryEscape(scope))
	// send request and wait for either a successful response or a timeout
	go s.sendOAuth2AcceptRequest(tokenRequestApplication, oAuth2URL)

	select {
	case <-tokenRequestApplication.granted:
		log.Debugf("token for scope %s got successfully extracted", scope)
	case <-time.After(Timeout):
		log.Warningf("no token redirect for scope %s occurred within %d seconds",
			scope, int(Timeout.Seconds()),
		)
	}
	log.Debugf("restoring previous CheckRedirect function")
	s.Client.CheckRedirect = tokenRequestApplication.sessionRedirect
	return tokenRequestApplication.token
}

// sendOAuth2AcceptRequest sets the CheckRedirect function of the client to our custom function
// and opens the OAuth2 URL. If the OAuth2 token fragments got found in the redirect it will cancel here
// else it will try to authorize the application and check for the token fragments once again
func (s *DeviantArtSession) sendOAuth2AcceptRequest(a *tokenRequestApplication, oAuth2URL string) {
	// save the previous CheckRedirect function for restoring it later and using it for checks in case
	// that the token fragments are not set in the request URL
	a.sessionRedirect = s.Client.CheckRedirect
	s.Client.CheckRedirect = a.checkRedirect

	// open the passed OAuth2 URL
	res, err := s.Get(oAuth2URL)
	raven.CheckError(err)

	// the application is authorized already if we already got the OAuth2 token, so no need to go further
	if a.token != nil {
		return
	}

	// we are currently in the authorization step since the previous redirect function didn't contain a token
	log.Info("application not authorized yet, sending authorization POST request")
	doc := s.GetDocument(res)
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
	// the custom redirect function is still active, so new check is executed here
	_, err = s.Post("https://www.deviantart.com/settings/authorize_app", values)
	raven.CheckError(err)
}
