package patreon

import "net/http"

// round tripper for custom user agent header
type userAgentRoundTripper struct {
	inner http.RoundTripper
	Agent string
}

// round tripper for required cloud flare headers
type cloudFlareRoundTripper struct {
	inner http.RoundTripper
}

// SetCloudFlareHeaders returns the round tripper to add required request headers on requests to pass CloudFlare checks
func (m *patreon) SetCloudFlareHeaders(inner http.RoundTripper) http.RoundTripper {
	return &cloudFlareRoundTripper{
		inner: inner,
	}
}

// SetUserAgent returns the round tripper to set a custom user agent on all requests
func (m *patreon) SetUserAgent(inner http.RoundTripper, userAgent string) http.RoundTripper {
	return &userAgentRoundTripper{
		inner: inner,
		Agent: userAgent,
	}
}

// RoundTrip adds the custom user agent to request headers
func (ug *userAgentRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("User-Agent", ug.Agent)

	if ug.inner == nil {
		return http.DefaultTransport.RoundTrip(r)
	}

	return ug.inner.RoundTrip(r)
}

// RoundTrip adds the required request headers to pass CloudFlare checks
func (ug *cloudFlareRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	r.Header.Set("Accept-Encoding", "gzip, deflate, br")
	r.Header.Set("Accept-Language", "en-US,en;q=0.5")

	if ug.inner == nil {
		return http.DefaultTransport.RoundTrip(r)
	}

	return ug.inner.RoundTrip(r)
}
