// Package session contains a default implementation of the session browser
package session

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"time"

	watcherHttp "github.com/DaRealFreak/watcher-go/internal/http"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/proxy"
	"golang.org/x/time/rate"
)

// DefaultSession is an extension to the implemented SessionInterface for HTTP sessions
type DefaultSession struct {
	watcherHttp.Session
	Client             *http.Client
	RateLimiter        *rate.Limiter
	ErrorHandlers      []watcherHttp.ErrorHandler
	MaxRetries         int
	MaxDownloadRetries int
	ctx                context.Context
}

// NewSession initializes a new session and sets all the required headers etc
func NewSession(moduleKey string, errorHandlers ...watcherHttp.ErrorHandler) *DefaultSession {
	jar, _ := cookiejar.New(nil)
	if len(errorHandlers) == 0 {
		errorHandlers = []watcherHttp.ErrorHandler{DefaultErrorHandler{}}
	}

	app := DefaultSession{
		Client: &http.Client{
			Jar:       jar,
			Transport: http.DefaultTransport,
		},
		ErrorHandlers:      errorHandlers,
		MaxRetries:         5,
		MaxDownloadRetries: 3,
		ctx:                context.Background(),
	}

	app.ModuleKey = moduleKey

	return &app
}

// Get sends a GET request, returns the occurred error if something went wrong even after multiple tries
func (s *DefaultSession) Get(uri string, errorHandlers ...watcherHttp.ErrorHandler) (response *http.Response, err error) {
	// access the passed url and return the data or the error which persisted multiple retries
	// post the request with the retries option
	for try := 1; try <= s.MaxRetries; try++ {
		s.ApplyRateLimit()

		log.WithField("module", s.ModuleKey).Debug(
			fmt.Sprintf("opening GET uri \"%s\" (try: %d)", uri, try),
		)

		response, err = s.Client.Get(uri)
		fatal := false
		// check session registered error handlers
		if err == nil {
			for _, errorHandler := range s.ErrorHandlers {
				if err, fatal = errorHandler.CheckResponse(response); err != nil {
					break
				}
			}
		}

		// check request registered error handlers
		if err == nil {
			for _, errorHandler := range errorHandlers {
				if err, fatal = errorHandler.CheckResponse(response); err != nil {
					break
				}
			}
		}

		// fatal error we can instantly return without retry
		if err != nil && fatal {
			return response, err
		}

		// no more errors, we can break out of the loop here
		if err == nil {
			break
		}

		// any other error falls into the retry clause
		time.Sleep(time.Duration(try+1) * time.Second)
	}

	return response, err
}

// Post sends a POST request, returns the occurred error if something went wrong even after multiple tries
func (s *DefaultSession) Post(uri string, data url.Values, errorHandlers ...watcherHttp.ErrorHandler) (response *http.Response, err error) {
	// post the request with the retries option
	for try := 1; try <= s.MaxRetries; try++ {
		s.ApplyRateLimit()

		log.WithField("module", s.ModuleKey).Debug(
			fmt.Sprintf("opening POST uri \"%s\" (try: %d)", uri, try),
		)

		response, err = s.Client.PostForm(uri, data)
		fatal := false
		// check session registered error handlers
		if err == nil {
			for _, errorHandler := range s.ErrorHandlers {
				if err, fatal = errorHandler.CheckResponse(response); err != nil {
					break
				}
			}
		}

		// check request registered error handlers
		if err == nil {
			for _, errorHandler := range errorHandlers {
				if err, fatal = errorHandler.CheckResponse(response); err != nil {
					break
				}
			}
		}

		// fatal error we can instantly return without retry
		if err != nil && fatal {
			return response, err
		}

		// no more errors, we can break out of the loop here
		if err == nil {
			break
		}

		// any other error falls into the retry clause
		time.Sleep(time.Duration(try+1) * time.Second)
	}

	return response, err
}

// DownloadFile tries to download the file, returns the occurred error if something went wrong even after multiple tries
func (s *DefaultSession) DownloadFile(filepath string, uri string, errorHandlers ...watcherHttp.ErrorHandler) (err error) {
	for try := 1; try <= s.MaxDownloadRetries; try++ {
		log.WithField("module", s.ModuleKey).Debug(
			fmt.Sprintf("downloading file: \"%s\" (uri: %s, try: %d)", filepath, uri, try),
		)

		err = s.tryDownloadFile(filepath, uri, errorHandlers...)
		if err != nil {
			// sleep if an error occurred
			time.Sleep(time.Duration(try+1) * time.Second)
		} else {
			// if no error occurred return nil
			return nil
		}
	}

	return err
}

// tryDownloadFile will try download an url to a local file.
// It's efficient because it will write as it downloads and not load the whole file into memory.
func (s *DefaultSession) tryDownloadFile(filepath string, uri string, errorHandlers ...watcherHttp.ErrorHandler) error {
	// retrieve the data
	resp, err := s.Get(uri, errorHandlers...)
	if err != nil {
		return err
	}

	defer raven.CheckClosure(resp.Body)

	// ensure the directory
	s.EnsureDownloadDirectory(filepath)

	if resp.StatusCode >= 400 {
		return StatusError{
			StatusCode: resp.StatusCode,
		}
	}

	// create the file
	out, createErr := os.Create(filepath)
	if createErr != nil {
		return createErr
	}

	defer raven.CheckClosure(out)

	// write the body to file
	written, copyErr := io.Copy(out, resp.Body)
	if copyErr != nil {
		return copyErr
	}

	// update parent folders access and modified times
	s.UpdateTreeFolderChangeTimes(filepath)

	// additional validation to compare sent headers with the written file
	for _, errorHandler := range s.ErrorHandlers {
		if err = errorHandler.CheckDownloadedFileForErrors(written, resp.Header); err != nil {
			return err
		}
	}

	for _, errorHandler := range errorHandlers {
		if err = errorHandler.CheckDownloadedFileForErrors(written, resp.Header); err != nil {
			return err
		}
	}

	return nil
}

// GetClient returns the used *http.Client, required f.e. to manually set cookies
func (s *DefaultSession) GetClient() *http.Client {
	return s.Client
}

// SetClient sets the used *http.Client in case we are routing the requests (for f.e. OAuth2 Authentications)
func (s *DefaultSession) SetClient(client *http.Client) {
	s.Client = client
}

// ApplyRateLimit waits for the leaky bucket to fill again
func (s *DefaultSession) ApplyRateLimit() {
	// if no rate limiter is defined we don't have to wait
	if s.RateLimiter != nil {
		// wait for request to stay within the rate limit
		err := s.RateLimiter.Wait(s.ctx)
		raven.CheckError(err)
	}
}

// SetProxy sets the current proxy for the client
func (s *DefaultSession) SetProxy(proxySettings *watcherHttp.ProxySettings) (err error) {
	if proxySettings != nil && proxySettings.Enable {
		log.WithField("module", s.ModuleKey).Infof(
			"setting proxy: [%s:%d]", proxySettings.Host, proxySettings.Port,
		)

		var proxyType string

		switch strings.ToUpper(proxySettings.Type) {
		case "SOCKS5":
			proxyType = "socks5"
		case "HTTP":
			proxyType = "http"
		case "HTTPS", "":
			proxyType = "https"
		default:
			return fmt.Errorf("unknown proxy type: %s", proxySettings.Type)
		}

		switch proxyType {
		case "socks5":
			auth := proxy.Auth{
				User:     proxySettings.Username,
				Password: proxySettings.Password,
			}

			dialer, err := proxy.SOCKS5(
				"tcp",
				fmt.Sprintf("%s:%d", proxySettings.Host, proxySettings.Port),
				&auth,
				proxy.Direct,
			)
			if err != nil {
				return err
			}

			s.GetClient().Transport = &http.Transport{Dial: dialer.Dial}
		default:
			var proxyURL *url.URL

			if proxySettings.Username != "" && proxySettings.Password != "" {
				proxyURL, _ = url.Parse(
					fmt.Sprintf(
						"%s://%s:%s@%s:%d",
						proxyType,
						url.QueryEscape(proxySettings.Username), url.QueryEscape(proxySettings.Password),
						url.QueryEscape(proxySettings.Host), proxySettings.Port,
					),
				)
			} else {
				proxyURL, _ = url.Parse(
					fmt.Sprintf(
						"%s://%s:%d",
						proxyType, url.QueryEscape(proxySettings.Host), proxySettings.Port,
					),
				)
			}

			s.GetClient().Transport = &http.Transport{Proxy: http.ProxyURL(proxyURL)}
		}
	}

	return nil
}
