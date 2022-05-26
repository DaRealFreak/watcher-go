package graphql_api

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/DaRealFreak/watcher-go/internal/http/session"
	log "github.com/sirupsen/logrus"
)

// CookieAuth is the authentication cookie which is set after a successful login
const CookieAuth = "auth_token"

// TwitterSession is a custom session to update cookies after responses for csrf tokens
type TwitterSession struct {
	session.DefaultSession
}

// NewTwitterSession initializes a new session
func NewTwitterSession(moduleKey string) *TwitterSession {
	return &TwitterSession{*session.NewSession(moduleKey)}
}

func (s *TwitterSession) Get(uri string) (response *http.Response, err error) {
	// access the passed url and return the data or the error which persisted multiple retries
	// post the request with the retries option
	for try := 1; try <= s.MaxRetries; try++ {
		s.ApplyRateLimit()

		log.WithField("module", s.ModuleKey).Debug(
			fmt.Sprintf("opening GET uri \"%s\" (try: %d)", uri, try),
		)

		response, err = s.Client.Get(uri)

		switch {
		case err == nil && response.StatusCode < 400:
			// if no error occurred and status code is okay to break out of the loop
			// 4xx & 5xx are client/server error codes, so we check for < 400
			return response, err

		case response.StatusCode == 403 && s.GetCSRFCookie() == "":
			// update of csrf token (expiration time of 3600 seconds)
			cookies := response.Cookies()
			for _, cookie := range cookies {
				if cookie.Name == "ct0" {
					requestUrl, _ := url.Parse("https://twitter.com/")
					s.GetClient().Jar.SetCookies(requestUrl, []*http.Cookie{cookie})
				}
			}

		default:
			tmp, _ := ioutil.ReadAll(response.Body)
			_ = tmp
			// any other error falls into the retry clause
			time.Sleep(time.Duration(try+1) * time.Second)
		}
	}

	return response, err
}

func (s *TwitterSession) GetCSRFCookie() string {
	requestUrl, _ := url.Parse("https://twitter.com/")
	cookies := s.GetClient().Jar.Cookies(requestUrl)
	for _, cookie := range cookies {
		if cookie.Name == "ct0" {
			return cookie.Value
		}
	}

	return ""
}
