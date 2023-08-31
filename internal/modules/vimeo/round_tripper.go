package vimeo

import (
	"net/http"
)

type vimeoRoundTripper struct {
	inner   http.RoundTripper
	referer string
}

// setDeviantArtHeaders returns the round tripper for the DeviantArt console API headers
func (m *vimeo) setVimeoHeaders(inner http.RoundTripper, referer string) http.RoundTripper {
	return &vimeoRoundTripper{
		inner:   inner,
		referer: referer,
	}
}

// RoundTrip adds the required request headers to pass server side checks of DeviantArt
func (rt *vimeoRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	if rt.referer != "" {
		r.Header.Set("Referer", rt.referer)
	}

	if r.URL.String() == "https://vimeo.com/_next/jwt" {
		r.Header.Set("X-Requested-With", "XMLHttpRequest")
	}

	if rt.inner == nil {
		return http.DefaultTransport.RoundTrip(r)
	}

	return rt.inner.RoundTrip(r)
}
