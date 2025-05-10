package graphql_api

import (
	"net/http"
	"strings"
	"time"

	"github.com/DaRealFreak/watcher-go/internal/http/session"
)

type DMCAError struct {
	error
}

func (e DMCAError) Error() string {
	return "content got most likely DMCAed"
}

type DeletedMediaError struct {
	error
}

func (e DeletedMediaError) Error() string {
	return "content got deleted"
}

type RateLimitError struct {
	error
}

func (e RateLimitError) Error() string {
	return "rate limit exceeded"
}

type CSRFError struct {
	error
}

func (e CSRFError) Error() string {
	return "invalid CSRF token"
}

type SessionTerminatedError struct {
	error
}

func (e SessionTerminatedError) Error() string {
	return "session got terminated"
}

type TwitterErrorHandler struct{}

func (e TwitterErrorHandler) CheckResponse(response *http.Response) (err error, fatal bool) {
	switch {
	case response.StatusCode == 401:
		// our session got likely terminated due to search rate limits being exceeded
		// so replace our auth token cookie with fallbacks if we have any
		return SessionTerminatedError{}, false
	case response.StatusCode == 429:
		// we are being rate limited
		// graphQL is not intended for public use so no rate limit is known, just sleep for an extended period of time
		// from my personal testing the limits are spliced into 15 minute intervals similar to their default API
		// https://developer.x.com/en/docs/twitter-api/rate-limits
		time.Sleep(5 * time.Minute)
		return RateLimitError{}, false
	case response.StatusCode == 403:
		if e.hasSetCookieHeader(response) {
			// retry if we first had to refresh/set cookies
			return CSRFError{}, false
		} else {
			return DMCAError{}, true
		}
	case response.StatusCode == 404:
		if strings.Contains(response.Request.URL.Hostname(), ".twimg.com") {
			return DeletedMediaError{}, true
		}
	}

	// fallback to default error handler
	defaultErrorHandler := session.DefaultErrorHandler{}
	return defaultErrorHandler.CheckResponse(response)
}

func (e TwitterErrorHandler) CheckDownloadedFileForErrors(writtenSize int64, responseHeader http.Header) error {
	// fallback to default error handler
	defaultErrorHandler := session.DefaultErrorHandler{}
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
