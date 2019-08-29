package session

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/DaRealFreak/watcher-go/pkg/http/session"
	log "github.com/sirupsen/logrus"
)

// DeviantArtSession is an extension of the DefaultSession, handling the OAuth2 Token process and POST requests
type DeviantArtSession struct {
	*session.DefaultSession
	TokenStore *TokenStore
}

// NewSession returns an initialized DeviantArtSession
func NewSession() *DeviantArtSession {
	return &DeviantArtSession{
		DefaultSession: session.NewSession(),
		TokenStore:     NewTokenStore(),
	}
}

// Post sends normal POST requests without any API scope/token contrary to the APIPost function
// it will try multiple times to successfully retrieve the http Response, if the error persists it will return the error
func (s *DeviantArtSession) Post(uri string, data url.Values) (response *http.Response, err error) {
	return s.post(uri, data, "")
}

// post is the internal POST functionality with the scope attached for OAuth2 Token refreshes
// if we receive an "Not Authorized" (401) status code
func (s *DeviantArtSession) post(uri string, data url.Values, scope string) (response *http.Response, err error) {
	// post the request with the retries option
	for try := 1; try <= s.MaxRetries; try++ {
		s.ApplyRateLimit()

		log.Debug(fmt.Sprintf("opening POST uri \"%s\" (try: %d)", uri, try))
		response, err = s.Client.PostForm(uri, data)
		switch {
		case err == nil && response.StatusCode < 400:
			// if no error occurred and status code is okay too break out of the loop
			// 4xx & 5xx are client/server error codes, so we check for < 400
			return response, err
		case err == nil && (response.StatusCode == 401 || response.StatusCode == 403) && scope != "":
			// on 401 or 403 we try to refresh our OAuth2 Token for the scope and try it again
			log.Infof("status code %d, refreshing OAuth2 Token", response.StatusCode)
			if s.RefreshOAuth2Token(scope) {
				data.Set("access_token", s.TokenStore.GetToken(scope).AccessToken)
				return s.post(uri, data, scope)
			}
			time.Sleep(time.Duration(try+1) * time.Second)
		default:
			// any other error falls into the retry clause
			time.Sleep(time.Duration(try+1) * time.Second)
		}
	}
	return response, err
}

// APIPost handles the OAuth2 Token for the POST request including refresh for API requests
func (s *DeviantArtSession) APIPost(uri string, data url.Values, scope string) (response *http.Response, err error) {
	// refresh OAuth2 Token if not set, return error if token couldn't get retrieved

	if !s.TokenStore.HasToken(scope) {
		if !s.RefreshOAuth2Token(scope) {
			return nil,
				fmt.Errorf(
					"could not retrieve the OAuth2 Token from the API for scope %s",
					scope,
				)
		}
	}
	// add the access token if not already set
	if data.Get("access_token") == "" {
		data.Set("access_token", s.TokenStore.GetToken(scope).AccessToken)
	}
	// handle the request now as a normal POST request
	return s.post(uri, data, scope)
}

// APIGet handles the OAuth2 Token for the GET request including refresh for API requests
func (s *DeviantArtSession) APIGet(uri string, scope string) (response *http.Response, err error) {
	// refresh OAuth2 Token if not set, return error if token couldn't get retrieved
	if !s.TokenStore.HasToken(scope) {
		if !s.RefreshOAuth2Token(scope) {
			return nil,
				fmt.Errorf(
					"could not retrieve the OAuth2 Token from the API for scope %s",
					scope,
				)
		}
	}

	apiURL, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	values := apiURL.Query()
	// add the access token if not already set
	if values.Get("access_token") == "" {
		values.Set("access_token", s.TokenStore.GetToken(scope).AccessToken)
	}
	apiURL.RawQuery = values.Encode()
	// handle the request now as a normal POST request
	return s.get(apiURL.String(), scope)
}

// Get sends normal GET requests without any API scope/token contrary to the APIGet function
// it will try multiple times to successfully retrieve the http Response, if the error persists it will return the error
func (s *DeviantArtSession) Get(uri string) (response *http.Response, err error) {
	return s.get(uri, "")
}

// get is the internal GET functionality with the scope attached for OAuth2 Token refreshes
// if we receive an "Not Authorized" (401) status code
func (s *DeviantArtSession) get(uri string, scope string) (response *http.Response, err error) {
	// post the request with the retries option
	for try := 1; try <= s.MaxRetries; try++ {
		s.ApplyRateLimit()

		log.Debug(fmt.Sprintf("opening GET uri \"%s\" (try: %d)", uri, try))
		response, err = s.Client.Get(uri)
		switch {
		case err == nil && response.StatusCode < 400:
			// if no error occurred and status code is okay too break out of the loop
			// 4xx & 5xx are client/server error codes, so we check for < 400
			return response, err
		case err == nil && (response.StatusCode == 401 || response.StatusCode == 403) && scope != "":
			// on 401 or 403 we try to refresh our OAuth2 Token for the scope and try it again
			log.Infof("status code %d, refreshing OAuth2 Token", response.StatusCode)
			if s.RefreshOAuth2Token(scope) {
				// replace access_token fragment with new token
				parsedURI, err := url.Parse(uri)
				if err != nil {
					return nil, err
				}
				values := parsedURI.Query()
				values.Set("access_token", s.TokenStore.GetToken(scope).AccessToken)
				parsedURI.RawQuery = values.Encode()
				return s.get(parsedURI.String(), scope)
			}
			time.Sleep(time.Duration(try+1) * time.Second)
		default:
			// any other error falls into the retry clause
			time.Sleep(time.Duration(try+1) * time.Second)
		}
	}
	return response, err
}

// RefreshOAuth2Token tries to retrieve the OAuth2 Token from the API for passed scope, returns the success as boolean
func (s *DeviantArtSession) RefreshOAuth2Token(scope string) (success bool) {
	s.TokenStore.SetToken(scope, s.retrieveOAuth2Token(scope))
	return s.TokenStore.GetToken(scope) != nil
}
