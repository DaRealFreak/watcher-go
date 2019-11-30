package jinjamodoki

import "net/http"

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
	r.Header.Set("Referer", "https://gs-uploader.jinja-modoki.com/")

	if m.inner == nil {
		return http.DefaultTransport.RoundTrip(r)
	}

	return m.inner.RoundTrip(r)
}
