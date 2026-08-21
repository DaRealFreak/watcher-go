package fourchan

import (
	"net/url"
	"testing"

	watcherHttp "github.com/DaRealFreak/watcher-go/internal/http"
	"github.com/DaRealFreak/watcher-go/internal/http/tls_session"
	"github.com/DaRealFreak/watcher-go/internal/models"
)

// fakeDbIO is a minimal DatabaseInterface serving a fixed set of cookies. Every other
// method panics via the embedded nil interface to surface unexpected usage.
type fakeDbIO struct {
	models.DatabaseInterface
	cookies []*models.Cookie
}

func (f *fakeDbIO) GetAllCookies(_ models.ModuleInterface) []*models.Cookie {
	return f.cookies
}

// newCookieTestModule builds a 4chan module with a real TLS session (so the cookie jar is
// the real one) and a database serving the Cloudflare cookies harvested from a browser.
func newCookieTestModule() (*fourChan, *tls_session.TlsClientSession) {
	m := NewBareModule().ModuleInterface.(*fourChan)
	session := tls_session.NewTlsClientSession(m.Key)
	session.SetClient(m.newHttpClient(session.Jar))
	m.Session = session
	m.DbIO = &fakeDbIO{cookies: []*models.Cookie{
		{Name: "cf_clearance", Value: "clearance-value"},
		{Name: "foolfuuka_99b1d8_csrf_token", Value: "csrf-value"},
	}}

	return m, session
}

// cookieValue returns the value of the named cookie in the passed set, or "" when absent.
func cookieValue(m *fourChan, rawURL string, name string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}

	for _, cookie := range m.Session.GetCookies(parsed) {
		if cookie.Name == name {
			return cookie.Value
		}
	}

	return ""
}

// TestSetCookiesAppliesArchiveHost is the regression test for the Cloudflare cookies never
// being sent: the cookies table keys rows by module only, so the generic Module.SetCookies
// applied them to the module key host (4chan.org) alone - while cf_clearance and the
// foolfuuka_* cookies belong to desuarchive.org, the origin actually behind Cloudflare.
func TestSetCookiesAppliesArchiveHost(t *testing.T) {
	m, _ := newCookieTestModule()

	m.SetCookies()

	if got := cookieValue(m, "https://desuarchive.org", "cf_clearance"); got != "clearance-value" {
		t.Errorf("cf_clearance for desuarchive.org = %q, want %q", got, "clearance-value")
	}
	if got := cookieValue(m, "https://desuarchive.org", "foolfuuka_99b1d8_csrf_token"); got != "csrf-value" {
		t.Errorf("foolfuuka csrf token for desuarchive.org = %q, want %q", got, "csrf-value")
	}
}

// TestSetCookiesKeepsModuleKeyHost guards the archive host addition against regressing the
// generic behaviour for the module key host.
func TestSetCookiesKeepsModuleKeyHost(t *testing.T) {
	m, _ := newCookieTestModule()

	m.SetCookies()

	if got := cookieValue(m, "https://4chan.org", "cf_clearance"); got != "clearance-value" {
		t.Errorf("cf_clearance for 4chan.org = %q, want %q", got, "clearance-value")
	}
}

// TestProxySessionsShareClearanceCookie locks in that the multi-proxy download pool sends
// the same Cloudflare clearance as the main session. The pool sessions are built on the
// main session's jar, so cookies set once after initialization reach all of them - which
// is what lets a single browser-harvested cf_clearance serve every proxy.
func TestProxySessionsShareClearanceCookie(t *testing.T) {
	m, mainSession := newCookieTestModule()
	m.settings.LoopProxies = []watcherHttp.ProxySettings{
		{Host: "proxy-a", Port: 1080, Type: "socks5", Enable: true},
		{Host: "proxy-b", Port: 1080, Type: "socks5", Enable: true},
		{Host: "proxy-c", Port: 1080, Type: "socks5", Enable: false},
	}

	// initialization happens before the cookies are loaded, mirroring InitializeModule
	// running ahead of SetCookies in Module.Load
	m.initializeProxySessions()
	m.SetCookies()

	if len(m.proxies) != 2 {
		t.Fatalf("initialized %d proxy sessions, want 2 (the disabled proxy is skipped)", len(m.proxies))
	}

	archiveURL, err := url.Parse("https://desuarchive.org")
	if err != nil {
		t.Fatalf("failed to parse the archive url: %v", err)
	}

	for _, proxy := range m.proxies {
		if proxy.session.Jar != mainSession.Jar {
			t.Errorf("proxy session %q does not share the main session's cookie jar", proxy.proxy.Host)
			continue
		}

		var found string
		for _, cookie := range proxy.session.GetCookies(archiveURL) {
			if cookie.Name == "cf_clearance" {
				found = cookie.Value
			}
		}

		if found != "clearance-value" {
			t.Errorf("proxy session %q sends cf_clearance %q, want %q", proxy.proxy.Host, found, "clearance-value")
		}
	}
}

// TestInitializeProxySessionsDoesNotStackUp locks in that a repeated initialization
// rebuilds the pool instead of appending a second set of sessions to it.
func TestInitializeProxySessionsDoesNotStackUp(t *testing.T) {
	m, _ := newCookieTestModule()
	m.settings.LoopProxies = []watcherHttp.ProxySettings{
		{Host: "proxy-a", Port: 1080, Type: "socks5", Enable: true},
	}

	m.initializeProxySessions()
	m.initializeProxySessions()

	if len(m.proxies) != 1 {
		t.Errorf("initialized %d proxy sessions after two calls, want 1", len(m.proxies))
	}
}
