package graphql_api

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"time"

	"github.com/DaRealFreak/watcher-go/internal/http/session"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	log "github.com/sirupsen/logrus"
)

// CookieAuth is the authentication cookie which is set after a successful login
const CookieAuth = "auth_token"

// TwitterSession is a custom session to update cookies after responses for csrf tokens
type TwitterSession struct {
	session.DefaultSession
}

type DMCAError struct {
	error
}

func (e DMCAError) Error() string {
	return "content got most likely DMCAed"
}

type RateLimitError struct {
	error
}

func (e RateLimitError) Error() string {
	return "rate limit exceeded"
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

		case response.StatusCode == 429:
			// we are being rate limited
			// graphQL is not intended for public use so no rate limit is known, just sleep for an extended period of time
			err = RateLimitError{}
			time.Sleep(time.Duration(try+1) * 20 * time.Second)
			break

		case response.StatusCode == 403 && !regexp.MustCompile(`https://twitter.com/i/api/graphql.*`).MatchString(uri):
			return response, DMCAError{}

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
			// any other error falls into the retry clause
			time.Sleep(time.Duration(try+1) * time.Second)
		}
	}

	return response, err
}

func (s *TwitterSession) DownloadFile(filepath string, uri string) (err error) {
	for try := 1; try <= s.MaxRetries; try++ {
		log.WithField("module", s.ModuleKey).Debug(
			fmt.Sprintf("downloading file: \"%s\" (uri: %s, try: %d)", filepath, uri, try),
		)

		err = s.tryDownloadFile(filepath, uri)
		switch err.(type) {
		case DMCAError:
			return err
		case nil:
			return nil
		default:
			time.Sleep(time.Duration(try+1) * time.Second)
		}
	}

	return err
}

// tryDownloadFile will try download an url to a local file.
// It's efficient because it will write as it downloads and not load the whole file into memory.
func (s *TwitterSession) tryDownloadFile(filepath string, uri string) error {
	// retrieve the data
	resp, err := s.Get(uri)
	if err != nil {
		return err
	}

	defer raven.CheckClosure(resp.Body)

	// ensure the directory
	s.EnsureDownloadDirectory(filepath)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("unexpected returned status code: %d", resp.StatusCode)
	}

	// create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}

	defer raven.CheckClosure(out)

	// write the body to file
	written, err := io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	// update parent folders access and modified times
	s.UpdateTreeFolderChangeTimes(filepath)

	// additional validation to compare sent headers with the written file
	return s.CheckDownloadedFileForErrors(written, resp.Header)
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
