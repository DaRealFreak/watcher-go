package jinjamodoki

import (
	http "github.com/bogdanfinn/fhttp"
	"net/url"
	"strings"

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

func (m *jinjaModoki) get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Origin", "https://gs-uploader.jinja-modoki.com")
	req.Header.Set("Referer", "https://gs-uploader.jinja-modoki.com/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:138.0) Gecko/20100101 Firefox/138.0")

	return m.Session.Do(req)
}

func (m *jinjaModoki) post(url string, data url.Values) (*http.Response, error) {
	formBody := data.Encode()
	req, err := http.NewRequest("POST", url, strings.NewReader(formBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Origin", "https://gs-uploader.jinja-modoki.com")
	req.Header.Set("Referer", "https://gs-uploader.jinja-modoki.com/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:138.0) Gecko/20100101 Firefox/138.0")

	return m.Session.Do(req)
}
