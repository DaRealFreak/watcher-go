package jinjamodoki

import (
	"net/http"

	browser "github.com/EDDYCJY/fake-useragent"
)

// round tripper for custom user agent header
type refererRoundTripper struct {
	inner http.RoundTripper
}

// SetReferer returns the round tripper to add required request headers on requests to pass CloudFlare checks
func (m *jinjaModoki) SetReferer(inner http.RoundTripper) http.RoundTripper {
	return &refererRoundTripper{
		inner: inner,
	}
}

// RoundTrip adds the custom user agent to request headers
func (m *refererRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	r.Header.Set("Accept-Language", "en-US,en;q=0.5")
	r.Header.Set("Origin", "https://gs-uploader.jinja-modoki.com")
	r.Header.Set("Referer", "https://gs-uploader.jinja-modoki.com/")
	r.Header.Set("User-Agent", browser.Firefox())

	if m.inner == nil {
		return http.DefaultTransport.RoundTrip(r)
	}

	return m.inner.RoundTrip(r)
}
