package session

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"time"

	"github.com/DaRealFreak/watcher-go/pkg/raven"

	watcherHttp "github.com/DaRealFreak/watcher-go/pkg/http"
	log "github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

// DefaultSession is an extension to the implemented SessionInterface for HTTP sessions
type DefaultSession struct {
	watcherHttp.Session
	Client      *http.Client
	RateLimiter *rate.Limiter
	MaxRetries  int
	ctx         context.Context
}

// NewSession initializes a new session and sets all the required headers etc
func NewSession() *DefaultSession {
	jar, _ := cookiejar.New(nil)

	app := DefaultSession{
		Client:     &http.Client{Jar: jar},
		MaxRetries: 5,
		ctx:        context.Background(),
	}
	return &app
}

// Get sends a GET request, returns the occurred error if something went wrong even after multiple tries
func (s *DefaultSession) Get(uri string) (response *http.Response, err error) {
	// access the passed url and return the data or the error which persisted multiple retries
	// post the request with the retries option
	for try := 1; try <= s.MaxRetries; try++ {
		s.applyRateLimit()

		log.Debug(fmt.Sprintf("opening GET uri \"%s\" (try: %d)", uri, try))
		response, err = s.Client.Get(uri)
		// if no error occurred break out of the loop
		if err == nil {
			break
		} else {
			time.Sleep(time.Duration(try+1) * time.Second)
		}
	}
	return response, err
}

// Post sends a POST request, returns the occurred error if something went wrong even after multiple tries
func (s *DefaultSession) Post(uri string, data url.Values) (response *http.Response, err error) {
	// post the request with the retries option
	for try := 1; try <= s.MaxRetries; try++ {
		s.applyRateLimit()

		log.Debug(fmt.Sprintf("opening POST uri \"%s\" (try: %d)", uri, try))
		response, err = s.Client.PostForm(uri, data)
		// if no error occurred break out of the loop
		if err == nil {
			break
		} else {
			time.Sleep(time.Duration(try+1) * time.Second)
		}
	}
	return response, err
}

// DownloadFile tries to download the file, returns the occurred error if something went wrong even after multiple tries
func (s *DefaultSession) DownloadFile(filepath string, uri string) (err error) {
	for try := 1; try <= s.MaxRetries; try++ {
		log.Info(fmt.Sprintf("downloading file: \"%s\" (uri: %s, try: %d)", filepath, uri, try))
		err = s.tryDownloadFile(filepath, uri)
		// if no error occurred return nil
		if err == nil {
			return
		}
		time.Sleep(time.Duration(try+1) * time.Second)
	}
	return err
}

// tryDownloadFile will try download a url to a local file.
// It's efficient because it will write as it downloads and not load the whole file into memory.
func (s *DefaultSession) tryDownloadFile(filepath string, uri string) error {
	// retrieve the data
	resp, err := s.Get(uri)
	if err != nil {
		return err
	}
	defer raven.CheckReadCloser(resp.Body)

	// ensure the directory
	s.EnsureDownloadDirectory(filepath)

	// create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer raven.CheckReadCloser(out)

	// write the body to file
	written, err := io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	// additional validation to compare sent headers with the written file
	err = s.CheckDownloadedFileForErrors(written, resp.Header)
	return err
}

// GetClient returns the used *http.Client, required f.e. to manually set cookies
func (s *DefaultSession) GetClient() *http.Client {
	return s.Client
}

// applyRateLimit waits for the leaky bucket to fill again
func (s *DefaultSession) applyRateLimit() {
	// if no rate limiter is defined we don't have to wait
	if s.RateLimiter != nil {
		// wait for request to stay within the rate limit
		err := s.RateLimiter.Wait(s.ctx)
		raven.CheckError(err)
	}
}