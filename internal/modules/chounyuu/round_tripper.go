package chounyuu

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/modules/chounyuu/api"

	browser "github.com/EDDYCJY/fake-useragent"
)

// round tripper for custom user agent header
type refererRoundTripper struct {
	inner http.RoundTripper
}

// SetReferer returns the round tripper to add required request headers on requests to pass CloudFlare checks
func (m *chounyuu) SetReferer(inner http.RoundTripper) http.RoundTripper {
	return &refererRoundTripper{
		inner: inner,
	}
}

// RoundTrip adds the custom user agent to request headers
func (m *refererRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("Accept-Language", "en-US,en;q=0.5")

	domain := api.ChounyuuDomain
	if strings.Contains(r.URL.Host, api.SuperFutaDomain) {
		domain = api.SuperFutaDomain
	}

	r.Header.Set("Origin", fmt.Sprintf("https://g.%s", domain))
	r.Header.Set("Referer", fmt.Sprintf("https://g.%s", domain))
	r.Header.Set("User-Agent", browser.Firefox())

	if m.inner == nil {
		return http.DefaultTransport.RoundTrip(r)
	}

	return m.inner.RoundTrip(r)
}
