package fourchan

import (
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tls_client "github.com/bogdanfinn/tls-client"
)

// TestUserAgentPrefersConfiguredValue locks in that a configured cloudflare.user_agent
// wins, it has to match the browser the cf_clearance cookie got harvested from.
func TestUserAgentPrefersConfiguredValue(t *testing.T) {
	m := NewBareModule().ModuleInterface.(*fourChan)
	m.settings.Cloudflare.UserAgent = "Mozilla/5.0 (X11; Linux x86_64; rv:128.0) Gecko/20100101 Firefox/128.0"

	if got := m.userAgent(); got != m.settings.Cloudflare.UserAgent {
		t.Errorf("userAgent() = %q, want the configured value %q", got, m.settings.Cloudflare.UserAgent)
	}
}

// TestUserAgentFallsBackToBrowserAgent locks in that an unset cloudflare.user_agent still
// produces a browser User-Agent instead of leaving it empty. Cloudflare rejects both an
// empty and the "Go-http-client/2.0" agent fhttp falls back to.
func TestUserAgentFallsBackToBrowserAgent(t *testing.T) {
	m := NewBareModule().ModuleInterface.(*fourChan)

	got := m.userAgent()
	if got == "" {
		t.Fatal("userAgent() returned an empty string, cloudflare rejects requests without a User-Agent")
	}
	if !strings.Contains(got, "Firefox") {
		t.Errorf("userAgent() = %q, want a Firefox agent matching the Firefox_147 client profile", got)
	}
}

// TestNewHttpClientSendsBrowserHeaders is the regression test for the Cloudflare 403s on
// desuarchive: the module used the plain tls_session client, which sends no User-Agent, so
// fhttp's HTTP/2 transport filled in "Go-http-client/2.0" and Cloudflare refused the
// request no matter which cf_clearance cookie the jar held. The default headers have to
// reach the wire for a request that brings none of its own, which is every request path
// the module uses.
func TestNewHttpClientSendsBrowserHeaders(t *testing.T) {
	var (
		gotUserAgent string
		gotAccept    string
	)

	server := httptest.NewServer(stdhttp.HandlerFunc(func(_ stdhttp.ResponseWriter, r *stdhttp.Request) {
		gotUserAgent = r.Header.Get("User-Agent")
		gotAccept = r.Header.Get("Accept")
	}))
	defer server.Close()

	m := NewBareModule().ModuleInterface.(*fourChan)
	client := m.newHttpClient(tls_client.NewCookieJar())
	if client == nil {
		t.Fatal("newHttpClient returned nil")
	}

	res, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = res.Body.Close() }()

	if strings.Contains(gotUserAgent, "Go-http-client") {
		t.Errorf("sent User-Agent %q, cloudflare rejects the fhttp default agent", gotUserAgent)
	}
	if gotUserAgent != defaultUserAgent {
		t.Errorf("sent User-Agent %q, want %q", gotUserAgent, defaultUserAgent)
	}
	if !strings.Contains(gotAccept, "text/html") {
		t.Errorf("sent Accept %q, want a browser document Accept header", gotAccept)
	}
}
