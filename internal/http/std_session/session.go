package std_session

import (
	"context"
	"fmt"
	watcherHttp "github.com/DaRealFreak/watcher-go/internal/http"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/proxy"
	"golang.org/x/time/rate"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"time"
)

// StdClientSession is an extension to the implemented StdClientSessionInterface for HTTP sessions
type StdClientSession struct {
	watcherHttp.StdClientSession
	Client             *http.Client
	RateLimiter        *rate.Limiter
	ErrorHandlers      []watcherHttp.StdClientErrorHandler
	MaxRetries         int
	MaxDownloadRetries int
	ctx                context.Context
}

// NewStdClientSession initializes a new session and sets all the required headers etc
func NewStdClientSession(moduleKey string, errorHandlers ...watcherHttp.StdClientErrorHandler) *StdClientSession {
	jar, _ := cookiejar.New(nil)
	if len(errorHandlers) == 0 {
		errorHandlers = []watcherHttp.StdClientErrorHandler{StdClientErrorHandler{}}
	}

	app := StdClientSession{
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
func (s *StdClientSession) Get(uri string, errorHandlers ...watcherHttp.StdClientErrorHandler) (response *http.Response, err error) {
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
func (s *StdClientSession) Post(uri string, data url.Values, errorHandlers ...watcherHttp.StdClientErrorHandler) (response *http.Response, err error) {
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
func (s *StdClientSession) Do(req *http.Request, errorHandlers ...watcherHttp.StdClientErrorHandler) (response *http.Response, err error) {
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
func (s *StdClientSession) DownloadFile(filepath string, uri string, errorHandlers ...watcherHttp.StdClientErrorHandler) (err error) {
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
func (s *StdClientSession) DownloadFileFromResponse(resp *http.Response, filepath string, errorHandlers ...watcherHttp.StdClientErrorHandler) (err error) {
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
func (s *StdClientSession) tryDownloadFile(filepath string, uri string, errorHandlers ...watcherHttp.StdClientErrorHandler) error {
	// retrieve the data
	resp, err := s.Get(uri, errorHandlers...)
	if err != nil {
		return err
	}

	return s.DownloadFileFromResponse(resp, filepath)
}

// GetClient returns the used *http.Client, required f.e. to manually set cookies
func (s *StdClientSession) GetClient() *http.Client {
	return s.Client
}

// SetClient sets the used *http.Client in case we are routing the requests (for f.e. OAuth2 Authentications)
func (s *StdClientSession) SetClient(client *http.Client) {
	s.Client = client
}

func (s *StdClientSession) GetCookies(u *url.URL) []*http.Cookie {
	return s.Client.Jar.Cookies(u)
}

func (s *StdClientSession) SetCookies(u *url.URL, cookies []*http.Cookie) {
	// set the cookies for the given URL
	s.Client.Jar.SetCookies(u, cookies)
}

func (s *StdClientSession) SetRateLimiter(rateLimiter *rate.Limiter) {
	s.RateLimiter = rateLimiter
}

// ApplyRateLimit waits for the leaky bucket to fill again
func (s *StdClientSession) ApplyRateLimit() {
	// if no rate limiter is defined, we don't have to wait
	if s.RateLimiter != nil {
		// wait for the request to stay within the rate limit
		err := s.RateLimiter.Wait(s.ctx)
		raven.CheckError(err)
	}
}

// SetProxy sets the current proxy for the client
func (s *StdClientSession) SetProxy(ps *watcherHttp.ProxySettings) error {
	// reset to default transport (no Proxy, default DialContext)
	if ps == nil || !ps.Enable || ps.Host == "" {
		if orig, ok := s.Client.Transport.(*http.Transport); ok {
			tr := orig.Clone()
			tr.Proxy = nil
			tr.DialContext = (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext
			s.Client.Transport = tr
		} else {
			s.Client.Transport = http.DefaultTransport
		}
		return nil
	}

	// pick proxy type
	proxyType := strings.ToLower(ps.Type)
	if proxyType == "" {
		proxyType = "https"
	}

	log.WithField("module", s.ModuleKey).Infof(
		"setting proxy: [%s %s:%d]", proxyType, ps.Host, ps.Port,
	)

	switch proxyType {
	case "socks5":
		addr := fmt.Sprintf("%s:%d", ps.Host, ps.Port)
		var auth *proxy.Auth
		if ps.Username != "" || ps.Password != "" {
			auth = &proxy.Auth{
				User:     ps.Username,
				Password: ps.Password,
			}
		}
		dialer, err := proxy.SOCKS5("tcp", addr, auth, proxy.Direct)
		if err != nil {
			return fmt.Errorf("failed to create SOCKS5 dialer: %w", err)
		}

		// wrap into a DialContext
		dialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.Dial(network, addr)
		}

		// apply onto cloned transport
		var tr *http.Transport
		if orig, ok := s.Client.Transport.(*http.Transport); ok {
			tr = orig.Clone()
		} else {
			tr = http.DefaultTransport.(*http.Transport).Clone()
		}
		tr.DialContext = dialContext
		// clear any HTTP Proxy setting
		tr.Proxy = nil
		s.Client.Transport = tr

	case "http", "https":
		// build URL with optional basic auth
		var userInfo string
		if ps.Username != "" {
			if ps.Password != "" {
				userInfo = url.UserPassword(ps.Username, ps.Password).String() + "@"
			} else {
				userInfo = url.User(ps.Username).String() + "@"
			}
		}
		proxyStr := fmt.Sprintf("%s://%s%s:%d", proxyType, userInfo, ps.Host, ps.Port)
		parsed, err := url.Parse(proxyStr)
		if err != nil {
			return fmt.Errorf("invalid proxy URL %q: %w", proxyStr, err)
		}

		var tr *http.Transport
		if orig, ok := s.Client.Transport.(*http.Transport); ok {
			tr = orig.Clone()
		} else {
			tr = http.DefaultTransport.(*http.Transport).Clone()
		}
		tr.Proxy = http.ProxyURL(parsed)
		// ensure default dialer
		tr.DialContext = (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext
		s.Client.Transport = tr

	default:
		return fmt.Errorf("unknown proxy type: %s", ps.Type)
	}

	return nil
}
