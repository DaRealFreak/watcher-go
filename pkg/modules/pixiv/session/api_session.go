package session

import (
	"net/http"
	"net/url"
)

// APISession is the struct to extend our PixivSession for custom header values for the different API types
type APISession struct {
	*PixivSession
	headers map[string]string
}

// NewPublicAPISession returns a custom implementation for the PixivSession
// with updated header values to use the previous Public API with our current context
func (s *PixivSession) NewPublicAPISession() *APISession {
	return &APISession{
		PixivSession: s,
		headers: map[string]string{
			"Referer": "http://spapi.pixiv.net/",
		},
	}
}

// Get returns the normal GET response with our updated header values
func (s *APISession) Get(uri string) (res *http.Response, err error) {
	// create backup of the default API headers
	originalHeaders := make(map[string]string)
	for key, value := range s.API.headers {
		originalHeaders[key] = value
	}
	// append/overwrite our custom headers before using the default Get function
	for headerKey, headerValue := range s.headers {
		s.API.headers[headerKey] = headerValue
	}
	res, err = s.PixivSession.Get(uri)
	// restore the original headers after usage
	s.API.headers = originalHeaders
	return res, err
}

// Post returns the normal POST response with our updated header values
func (s *APISession) Post(uri string, data url.Values) (res *http.Response, err error) {
	// create backup of the default API headers
	originalHeaders := make(map[string]string)
	for key, value := range s.API.headers {
		originalHeaders[key] = value
	}
	// append/overwrite our custom headers before using the default Get function
	for headerKey, headerValue := range s.headers {
		s.API.headers[headerKey] = headerValue
	}
	res, err = s.PixivSession.Post(uri, data)
	// restore the original headers after usage
	s.API.headers = originalHeaders
	return res, err
}
