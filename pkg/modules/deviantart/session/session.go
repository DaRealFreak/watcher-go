package session

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/DaRealFreak/watcher-go/pkg/http/session"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

// DeviantArtSession is an extension of the DefaultSession, handling the OAuth2 Token process and POST requests
type DeviantArtSession struct {
	*session.DefaultSession
	AccessToken *oauth2.Token
}

// NewSession returns an initialized DeviantArtSession
func NewSession() *DeviantArtSession {
	return &DeviantArtSession{
		DefaultSession: session.NewSession(),
	}
}

// Post sends a POST request, returns the occurred error if something went wrong even after multiple tries
// if the returned status code is 401 we are trying to refresh the OAuth2 Token and try it again
func (s *DeviantArtSession) Post(uri string, data url.Values) (response *http.Response, err error) {
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
		case err == nil && response.StatusCode == 401:
			// on 401 we try to refresh our OAuth2 Token and try it again
			log.Infof("status code %d, refreshing OAuth2 Token", response.StatusCode)
			s.AccessToken = s.retrieveOAuth2Token()
			if s.AccessToken != nil {
				data.Set("access_token", s.AccessToken.AccessToken)
				return s.Post(uri, data)
			}
			time.Sleep(time.Duration(try+1) * time.Second)
		default:
			// any other error falls into the retry clause
			time.Sleep(time.Duration(try+1) * time.Second)
		}
	}
	return response, err
}

// APIPost attaches automatically the OAuth2 Token or tries to refresh it if not set
func (s *DeviantArtSession) APIPost(uri string, data url.Values) (response *http.Response, err error) {
	// refresh OAuth2 Token if not set, return error if token couldn't get retrieved
	if s.AccessToken == nil {
		if !s.RefreshOAuth2Token() {
			return nil, fmt.Errorf("could not retrieve the OAuth2 Token from the API")
		}
	}
	// add the access token if not already set
	if data.Get("access_token") == "" {
		data.Set("access_token", s.AccessToken.AccessToken)
	}
	// handle the request now as a normal POST request
	return s.Post(uri, data)
}

// RefreshOAuth2Token tries to retrieve the OAuth2 Token from the API, returns the success as boolean
func (s *DeviantArtSession) RefreshOAuth2Token() (success bool) {
	s.AccessToken = s.retrieveOAuth2Token()
	return s.AccessToken != nil
}
