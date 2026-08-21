package fourchan

import (
	"os"
	"strings"
	"testing"

	"github.com/DaRealFreak/watcher-go/internal/http/tls_session"
	"github.com/DaRealFreak/watcher-go/internal/models"
)

const (
	// EnvCfClearance is the environment variable holding the cf_clearance cookie value
	EnvCfClearance = "FOURCHAN_CF_CLEARANCE"
	// EnvUserAgent is the environment variable holding the user agent the cf_clearance
	// cookie got issued for, it has to match or cloudflare rejects the cookie
	EnvUserAgent = "FOURCHAN_USER_AGENT"
)

// skipWithoutCloudflareClearance skips the test if no cf_clearance cookie and matching user
// agent are configured. desuarchive answers every page request with a cloudflare challenge,
// and cloudflare validates the clearance cookie against both the IP and the User-Agent it
// got issued for - so the cookie has to come from a browser on the same exit the test runs
// from, which no CI can provide.
func skipWithoutCloudflareClearance(t *testing.T) {
	t.Helper()

	if os.Getenv(EnvCfClearance) == "" || os.Getenv(EnvUserAgent) == "" {
		t.Skipf("%s and %s are required, desuarchive answers page requests with a cloudflare challenge",
			EnvCfClearance, EnvUserAgent)
	}
}

// TestGetPageWithCloudflareClearance drives the real module page retrieval against
// desuarchive with a browser harvested clearance, covering the whole chain at once: the
// browser default headers on the session client, the stored cookie reaching the
// desuarchive.org host, and pages_without_proxy keeping the request on the exit the
// clearance belongs to. A 403 here means one of the three regressed.
func TestGetPageWithCloudflareClearance(t *testing.T) {
	skipWithoutCloudflareClearance(t)

	m := NewBareModule().ModuleInterface.(*fourChan)
	m.settings.Cloudflare.UserAgent = os.Getenv(EnvUserAgent)
	m.settings.Cloudflare.PagesWithoutProxy = true

	session := tls_session.NewTlsClientSession(m.Key)
	session.SetClient(m.newHttpClient(session.Jar))
	m.Session = session
	m.DbIO = &fakeDbIO{cookies: []*models.Cookie{
		{Name: "cf_clearance", Value: os.Getenv(EnvCfClearance)},
	}}

	m.SetCookies()

	res, err := m.getPage("https://desuarchive.org/d/search/subject/Giantess/")
	if err != nil {
		t.Fatalf("page retrieval failed: %v", err)
	}
	if res.StatusCode != 200 {
		t.Fatalf("page retrieval returned status %d, want 200", res.StatusCode)
	}

	html, htmlErr := m.Session.GetDocument(res).Html()
	if htmlErr != nil {
		t.Fatalf("failed to read the document: %v", htmlErr)
	}
	if strings.Contains(html, "Just a moment") {
		t.Fatal("received the cloudflare interstitial instead of the search page")
	}

	// the page has to be parseable, not merely non-403
	if threads := m.getThreads(html); len(threads) == 0 {
		t.Error("parsed 0 threads from the search page, the response is not the expected document")
	}
}
