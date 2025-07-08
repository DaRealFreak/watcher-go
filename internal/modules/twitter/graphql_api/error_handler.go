package graphql_api

import (
	"github.com/DaRealFreak/watcher-go/internal/http/tls_session"
	http "github.com/bogdanfinn/fhttp"
	"strings"
	"time"
)

type DMCAError struct {
}

func (e DMCAError) Error() string {
	return "content got most likely DMCAed"
}

type DeletedMediaError struct {
}

func (e DeletedMediaError) Error() string {
	return "content got deleted"
}

type RateLimitError struct {
}

func (e RateLimitError) Error() string {
	return "rate limit exceeded"
}

type CSRFError struct {
}

func (e CSRFError) Error() string {
	return "invalid CSRF token"
}

type SessionTerminatedError struct {
}

func (e SessionTerminatedError) Error() string {
	return "session got terminated"
}

type SessionRefreshError struct {
}

func (e SessionRefreshError) Error() string {
	return "session requires a refresh for x-transaction-id to be valid again"
}

type TwitterErrorHandler struct{}

func (e TwitterErrorHandler) CheckResponse(response *http.Response) (err error, fatal bool) {
	switch response.StatusCode {
	case 401:
		// our session got likely terminated due to search rate limits being exceeded
		// so replace our auth token cookie with fallbacks if we have any
		return SessionTerminatedError{}, false
	case 429:
		// we are being rate limited
		// graphQL is not intended for public use so no rate limit is known, just sleep for an extended period of time
		// from my personal testing the limits are spliced into 15 minute intervals similar to their default API
		// https://developer.x.com/en/docs/twitter-api/rate-limits
		time.Sleep(5 * time.Minute)
		return RateLimitError{}, false
	case 403:
		if e.hasSetCookieHeader(response) {
			// retry if we first had to refresh/set cookies
			return CSRFError{}, false
		} else {
			return DMCAError{}, true
		}
	case 404:
		if strings.Contains(response.Request.URL.Hostname(), ".twimg.com") {
			return DeletedMediaError{}, true
		} else if strings.Contains(response.Request.URL.String(), "/i/api/graphql/") {
			return SessionRefreshError{}, true
		}
	}

	// fallback to default error handler
	defaultErrorHandler := tls_session.TlsClientErrorHandler{}
	return defaultErrorHandler.CheckResponse(response)
}

func (e TwitterErrorHandler) CheckDownloadedFileForErrors(writtenSize int64, responseHeader http.Header) error {
	// fallback to default error handler
	defaultErrorHandler := tls_session.TlsClientErrorHandler{}
	return defaultErrorHandler.CheckDownloadedFileForErrors(writtenSize, responseHeader)
}

func (e TwitterErrorHandler) hasSetCookieHeader(response *http.Response) bool {
	for key := range response.Header {
		if key == "Set-Cookie" {
			return true
		}
	}
	return false
}

func (e TwitterErrorHandler) IsFatalError(_ error) bool {
	return false
}
