package napi

import (
	http "github.com/bogdanfinn/fhttp"
)

type deviantArtConsoleAPIRoundTripper struct {
	inner http.RoundTripper
}

// setDeviantArtHeaders returns the round tripper for the DeviantArt console API headers
func (a *DeviantartNAPI) setDeviantArtHeaders(inner http.RoundTripper) http.RoundTripper {
	return &deviantArtConsoleAPIRoundTripper{
		inner: inner,
	}
}

// RoundTrip adds the required request headers to pass server side checks of DeviantArt
func (rt *deviantArtConsoleAPIRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("Accept-Language", "en-US,en;q=0.5")

	if r.URL.Host == "www.deviantart.com" {
		r.Header.Set("Host", "https://www.deviantart.com")
		r.Header.Set("Referer", "https://www.deviantart.com")
		r.Header.Set("Origin", "https://www.deviantart.com")
	}

	if rt.inner == nil {
		return http.DefaultTransport.RoundTrip(r)
	}

	return rt.inner.RoundTrip(r)
}
