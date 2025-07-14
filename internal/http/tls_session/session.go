package tls_session

import (
	"context"
	"fmt"
	watcherHttp "github.com/DaRealFreak/watcher-go/internal/http"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
	log "github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
	"io"
	"net/url"
	"os"
	"strings"
	"time"
)

// TlsClientSession is an extension to the implemented TlsClientSessionInterface for HTTP sessions
type TlsClientSession struct {
	watcherHttp.TlsClientSession
	Client             tls_client.HttpClient
	RateLimiter        *rate.Limiter
	ErrorHandlers      []watcherHttp.TlsClientErrorHandler
	MaxRetries         int
	MaxDownloadRetries int
	ctx                context.Context
}

// NewTlsClientSession initializes a new session and sets all the required headers etc
func NewTlsClientSession(moduleKey string, errorHandlers ...watcherHttp.TlsClientErrorHandler) *TlsClientSession {
	jar := tls_client.NewCookieJar()
	if len(errorHandlers) == 0 {
		errorHandlers = []watcherHttp.TlsClientErrorHandler{TlsClientErrorHandler{}}
	}

	options := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(10 * 60),
		tls_client.WithClientProfile(profiles.Firefox_135),
		tls_client.WithRandomTLSExtensionOrder(),
		tls_client.WithCookieJar(jar),
	}

	client, _ := tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)

	app := TlsClientSession{
		Client:             client,
		ErrorHandlers:      errorHandlers,
		MaxRetries:         5,
		MaxDownloadRetries: 3,
		ctx:                context.Background(),
	}

	app.ModuleKey = moduleKey

	return &app
}

// Get sends a GET request, returns the occurred error if something went wrong even after multiple tries
func (s *TlsClientSession) Get(uri string, errorHandlers ...watcherHttp.TlsClientErrorHandler) (response *http.Response, err error) {
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
		} else {
			for _, errorHandler := range s.ErrorHandlers {
				if fatal = errorHandler.IsFatalError(err); fatal {
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
		} else {
			// if fatal is already true, we don't have to check the request error handlers
			if !fatal {
				for _, errorHandler := range errorHandlers {
					if fatal = errorHandler.IsFatalError(err); fatal {
						break
					}
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
func (s *TlsClientSession) Post(uri string, data url.Values, errorHandlers ...watcherHttp.TlsClientErrorHandler) (response *http.Response, err error) {
	formBody := data.Encode() // "k=v&x=y"
	for try := 1; try <= s.MaxRetries; try++ {
		s.ApplyRateLimit()

		log.WithField("module", s.ModuleKey).Debug(
			fmt.Sprintf("opening GET uri \"%s\" (try: %d)", uri, try),
		)

		// use the generic Post method on the TLS-Client interface:
		response, err = s.Client.Post(uri,
			"application/x-www-form-urlencoded",
			strings.NewReader(formBody),
		)
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
		time.Sleep(time.Duration(try+1) * time.Second)
	}
	return response, err
}

// Do function handles the passed request, returns the occurred error if something went wrong even after multiple tries
func (s *TlsClientSession) Do(req *http.Request, errorHandlers ...watcherHttp.TlsClientErrorHandler) (response *http.Response, err error) {
	for try := 1; try <= s.MaxRetries; try++ {
		s.ApplyRateLimit()

		log.WithField("module", s.ModuleKey).Debug(
			fmt.Sprintf("opening %s uri \"%s\" (try: %d)", req.Method, req.URL.String(), try),
		)

		// use the generic Post method on the TLS-Client interface:
		response, err = s.Client.Do(req)
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
		time.Sleep(time.Duration(try+1) * time.Second)
	}
	return response, err
}

// DownloadFile tries to download the file, returns the occurred error if something went wrong even after multiple tries
func (s *TlsClientSession) DownloadFile(filepath string, uri string, errorHandlers ...watcherHttp.TlsClientErrorHandler) (err error) {
	log.WithField("module", s.ModuleKey).Debug(
		fmt.Sprintf("downloading file: \"%s\" (uri: %s)", filepath, uri),
	)

	err = s.tryDownloadFile(filepath, uri, errorHandlers...)
	if err != nil {
		// try to clean up the failed file if it exists
		if _, statErr := os.Stat(filepath); statErr == nil {
			_ = os.Remove(filepath)
		}
	}

	return err
}

// DownloadFileFromResponse tries to download the file from the response, returns the occurred error if something went wrong even after multiple tries
func (s *TlsClientSession) DownloadFileFromResponse(resp *http.Response, filepath string, errorHandlers ...watcherHttp.TlsClientErrorHandler) (err error) {
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

// tryDownloadFile will try download an url to a local file.
// It's efficient because it will write as it downloads and not load the whole file into memory.
func (s *TlsClientSession) tryDownloadFile(filepath string, uri string, errorHandlers ...watcherHttp.TlsClientErrorHandler) error {
	// retrieve the data
	resp, err := s.Get(uri, errorHandlers...)
	if err != nil {
		return err
	}

	return s.DownloadFileFromResponse(resp, filepath)
}

// GetClient returns the used *http.Client, required f.e. to manually set cookies
func (s *TlsClientSession) GetClient() tls_client.HttpClient {
	return s.Client
}

// SetClient sets the used *http.Client in case we are routing the requests (for f.e. OAuth2 Authentications)
func (s *TlsClientSession) SetClient(client tls_client.HttpClient) {
	s.Client = client
}

func (s *TlsClientSession) GetCookies(u *url.URL) []*http.Cookie {
	return s.Client.GetCookies(u)
}

func (s *TlsClientSession) SetCookies(u *url.URL, cookies []*http.Cookie) {
	// set the cookies for the given URL
	s.Client.SetCookies(u, cookies)
}

func (s *TlsClientSession) SetRateLimiter(rateLimiter *rate.Limiter) {
	s.RateLimiter = rateLimiter
}

// ApplyRateLimit waits for the leaky bucket to fill again
func (s *TlsClientSession) ApplyRateLimit() {
	// if no rate limiter is defined, we don't have to wait
	if s.RateLimiter != nil {
		// wait for the request to stay within the rate limit
		err := s.RateLimiter.Wait(s.ctx)
		raven.CheckError(err)
	}
}

// SetProxy sets the current proxy for the client
func (s *TlsClientSession) SetProxy(ps *watcherHttp.ProxySettings) error {
	if ps == nil || !ps.Enable || ps.Host == "" {
		return s.Client.SetProxy("")
	}

	proxyType := strings.ToLower(ps.Type)
	if proxyType == "" {
		proxyType = "https"
	}

	auth := ""
	if ps.Username != "" && ps.Password != "" {
		auth = url.QueryEscape(ps.Username) + ":" +
			url.QueryEscape(ps.Password) + "@"
	}
	proxyURL := fmt.Sprintf(
		"%s://%s%s:%d",
		proxyType,
		auth,
		url.QueryEscape(ps.Host),
		ps.Port,
	)

	log.WithField("module", s.ModuleKey).Debug(
		fmt.Sprintf("setting proxy: %s", strings.Replace(proxyURL, auth, "****:****@", 1)),
	)

	return s.Client.SetProxy(proxyURL)
}
