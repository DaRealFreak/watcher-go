package fourchan

import (
	fhttp "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
)

// defaultUserAgent is a Firefox 147 desktop User-Agent matching the profiles.Firefox_147
// TLS fingerprint the module's sessions use. Without an explicit header fhttp's HTTP/2
// transport falls back to "Go-http-client/2.0", which Cloudflare rejects outright and
// which also invalidates a cf_clearance cookie issued to a real browser.
const defaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:147.0) Gecko/20100101 Firefox/147.0"

// userAgent returns the configured Cloudflare User-Agent, falling back to
// defaultUserAgent. Cloudflare validates a cf_clearance cookie against the User-Agent it
// got issued for, so this has to match the browser the cookie was harvested from.
func (m *fourChan) userAgent() string {
	if m.settings.Cloudflare.UserAgent != "" {
		return m.settings.Cloudflare.UserAgent
	}

	return defaultUserAgent
}

// defaultHeaders returns the browser headers every request of the module carries. The
// tls-client applies them to any request that doesn't bring headers of its own, which
// covers all four request paths the module uses: the page retrieval through the session,
// the raw client probes in threadRemoved and the multi-proxy download pool, and
// DownloadFile on the single-session download path.
func (m *fourChan) defaultHeaders() fhttp.Header {
	return fhttp.Header{
		"accept":                    {"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8"},
		"accept-language":           {"en-US,en;q=0.5"},
		"upgrade-insecure-requests": {"1"},
		"user-agent":                {m.userAgent()},
	}
}

// newHttpClient builds a tls-client on the passed cookie jar carrying the module's
// default headers. It mirrors the options of tls_session.NewTlsClientSessionWithJar,
// which offers no way to pass default headers, so the session gets this client injected
// through SetClient instead (same approach as the schalenetwork module).
func (m *fourChan) newHttpClient(jar tls_client.CookieJar) tls_client.HttpClient {
	client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(),
		tls_client.WithTimeoutSeconds(30*60),
		tls_client.WithClientProfile(profiles.Firefox_147),
		tls_client.WithRandomTLSExtensionOrder(),
		tls_client.WithCookieJar(jar),
		tls_client.WithDefaultHeaders(m.defaultHeaders()),
	)
	if err != nil {
		return nil
	}

	return client
}
