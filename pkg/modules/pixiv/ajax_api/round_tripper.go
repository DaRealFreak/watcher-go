package ajax_api

import (
	browser "github.com/EDDYCJY/fake-useragent"
	"net/http"
)

type pixivRoundTripper struct {
	inner http.RoundTripper
}

// SetPixivWebHeaders
func SetPixivWebHeaders(inner http.RoundTripper) http.RoundTripper {
	return &pixivRoundTripper{
		inner: inner,
	}
}

// RoundTrip adds the required request headers to pass CloudFlare checks
func (rt *pixivRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	r.Header.Set("Accept-Encoding", "gzip, deflate, br")
	r.Header.Set("Accept-Language", "en-US,en;q=0.5")
	r.Header.Set("User-Agent", browser.Firefox())
	r.Header.Set("Referer", "https://pixiv.net/")

	if rt.inner == nil {
		return http.DefaultTransport.RoundTrip(r)
	}

	return rt.inner.RoundTrip(r)
}
