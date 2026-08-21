package fourchan

import (
	"errors"
	"testing"

	"github.com/DaRealFreak/watcher-go/internal/http"
	"github.com/DaRealFreak/watcher-go/internal/http/tls_session"
	fhttp "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
)

// getResult scripts one fakeSession.Get outcome.
type getResult struct {
	statusCode int
	err        error
}

// fakeSession is a minimal TlsClientSessionInterface used to drive getPage without
// performing real network requests. Only Get, SetProxy and GetClient are exercised; any
// other method panics (via the embedded nil interface) to surface unexpected usage.
type fakeSession struct {
	http.TlsClientSessionInterface
	results      []getResult
	calls        int
	appliedProxy []string
	client       *fakeClient
}

func (f *fakeSession) Get(_ string, _ ...http.TlsClientErrorHandler) (*fhttp.Response, error) {
	result := f.results[f.calls]
	f.calls++

	return &fhttp.Response{StatusCode: result.statusCode}, result.err
}

func (f *fakeSession) GetClient() tls_client.HttpClient {
	return f.client
}

func (f *fakeSession) SetProxy(proxySettings *http.ProxySettings) error {
	if proxySettings == nil {
		f.appliedProxy = append(f.appliedProxy, "")
	} else {
		f.appliedProxy = append(f.appliedProxy, proxySettings.Host)
	}

	return nil
}

// newProxyTestModule builds a 4chan module in loop mode with the given proxies and a
// fake session for deterministic page-retrieval tests.
func newProxyTestModule(proxies []http.ProxySettings) (*fourChan, *fakeSession) {
	m := NewBareModule().ModuleInterface.(*fourChan)
	m.settings.Loop = true
	m.settings.LoopProxies = proxies

	fake := &fakeSession{}
	m.Session = fake

	return m, fake
}

func status403() error {
	return tls_session.StatusError{StatusCode: 403}
}

func TestSetProxyMethodSkipsEvictedAndWraps(t *testing.T) {
	proxies := []http.ProxySettings{
		{Host: "proxy-a", Port: 1080, Enable: true},
		{Host: "proxy-b", Port: 1080, Enable: true},
		{Host: "proxy-c", Port: 1080, Enable: true},
	}
	m, _ := newProxyTestModule(proxies)

	// evict the middle proxy; rotation must skip it and wrap back around
	m.evictedProxies[proxyKey(&proxies[1])] = true

	want := []string{"proxy-a", "proxy-c", "proxy-a"}
	for i, expected := range want {
		if err := m.setProxyMethod(); err != nil {
			t.Fatalf("rotation %d returned error: %v", i, err)
		}
		if got := proxies[m.ProxyLoopIndex].Host; got != expected {
			t.Errorf("rotation %d selected %q, want %q", i, got, expected)
		}
	}
}

func TestSetProxyMethodErrorsWhenAllEvicted(t *testing.T) {
	proxies := []http.ProxySettings{
		{Host: "proxy-a", Port: 1080, Enable: true},
		{Host: "proxy-b", Port: 1080, Enable: true},
	}
	m, _ := newProxyTestModule(proxies)
	m.evictedProxies[proxyKey(&proxies[0])] = true
	m.evictedProxies[proxyKey(&proxies[1])] = true

	if err := m.setProxyMethod(); err == nil {
		t.Fatal("expected an error when every loop proxy is evicted")
	}
}

func TestGetPageEvictsProxyOn403AndRetries(t *testing.T) {
	proxies := []http.ProxySettings{
		{Host: "proxy-a", Port: 1080, Enable: true},
		{Host: "proxy-b", Port: 1080, Enable: true},
	}
	m, fake := newProxyTestModule(proxies)
	fake.results = []getResult{
		{statusCode: 403, err: status403()}, // proxy-a fails
		{statusCode: 200, err: nil},         // proxy-b succeeds
	}

	res, err := m.getPage("https://desuarchive.org/d/search/subject/test/")
	if err != nil {
		t.Fatalf("expected success after eviction+retry, got: %v", err)
	}
	if res.StatusCode != 200 {
		t.Fatalf("expected status 200 from the retry, got %d", res.StatusCode)
	}
	if !m.evictedProxies[proxyKey(&proxies[0])] {
		t.Error("proxy-a should be evicted after its 403")
	}
	if m.evictedProxies[proxyKey(&proxies[1])] {
		t.Error("proxy-b should remain usable")
	}
	if want := []string{"proxy-a", "proxy-b"}; len(fake.appliedProxy) != 2 ||
		fake.appliedProxy[0] != want[0] || fake.appliedProxy[1] != want[1] {
		t.Errorf("unexpected rotation order: %v, want %v", fake.appliedProxy, want)
	}
}

func TestGetPageReturnsErrorWhenAllProxiesEvicted(t *testing.T) {
	proxies := []http.ProxySettings{
		{Host: "proxy-a", Port: 1080, Enable: true},
		{Host: "proxy-b", Port: 1080, Enable: true},
	}
	m, fake := newProxyTestModule(proxies)
	fake.results = []getResult{
		{statusCode: 403, err: status403()},
		{statusCode: 403, err: status403()},
	}

	_, err := m.getPage("https://desuarchive.org/d/search/subject/test/")
	if err == nil {
		t.Fatal("expected the 403 to propagate once all proxies are evicted")
	}

	var statusErr tls_session.StatusError
	if !errors.As(err, &statusErr) || statusErr.StatusCode != 403 {
		t.Fatalf("expected a 403 StatusError, got: %v", err)
	}
	if !m.evictedProxies[proxyKey(&proxies[0])] || !m.evictedProxies[proxyKey(&proxies[1])] {
		t.Error("both proxies should be evicted after returning 403")
	}
}

func TestGetPageNonLoopFallsBackToPlainGet(t *testing.T) {
	m, fake := newProxyTestModule(nil)
	m.settings.Loop = false
	fake.results = []getResult{{statusCode: 200, err: nil}}

	res, err := m.getPage("https://desuarchive.org/d/search/subject/test/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	// no proxy rotation should occur outside loop mode
	if len(fake.appliedProxy) != 0 {
		t.Errorf("expected no SetProxy calls in non-loop mode, got %v", fake.appliedProxy)
	}
}

// TestGetPageWithoutProxyKeepsDirectConnection is the regression test for the Cloudflare
// 403s that survived the User-Agent and cookie fix: rotating the loop proxies presents one
// stored cf_clearance from exits it was never issued for. Verified against the live site by
// replaying a single clearance - 200 from the exit that solved the challenge, 403 with
// "cf-mitigated: challenge" from three different proxy exits. With pages_without_proxy the
// page requests must stay on the direct connection and must not rotate.
func TestGetPageWithoutProxyKeepsDirectConnection(t *testing.T) {
	proxies := []http.ProxySettings{
		{Host: "proxy-a", Port: 1080, Enable: true},
		{Host: "proxy-b", Port: 1080, Enable: true},
	}
	m, fake := newProxyTestModule(proxies)
	m.settings.Cloudflare.PagesWithoutProxy = true
	fake.results = []getResult{{statusCode: 200, err: nil}}

	res, err := m.getPage("https://desuarchive.org/d/search/subject/test/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	if len(fake.appliedProxy) != 0 {
		t.Errorf("page request rotated onto %v, want no proxy rotation at all", fake.appliedProxy)
	}
}

// TestGetPageWithoutProxyDoesNotEvictOn403 locks in that a 403 propagates instead of
// burning through the proxy pool. Without a proxy in play a 403 means the stored clearance
// no longer matches, and no other exit can satisfy it either.
func TestGetPageWithoutProxyDoesNotEvictOn403(t *testing.T) {
	proxies := []http.ProxySettings{
		{Host: "proxy-a", Port: 1080, Enable: true},
		{Host: "proxy-b", Port: 1080, Enable: true},
	}
	m, fake := newProxyTestModule(proxies)
	m.settings.Cloudflare.PagesWithoutProxy = true
	fake.results = []getResult{{statusCode: 403, err: status403()}}

	_, err := m.getPage("https://desuarchive.org/d/search/subject/test/")
	if err == nil {
		t.Fatal("expected the 403 to propagate")
	}

	var statusErr tls_session.StatusError
	if !errors.As(err, &statusErr) || statusErr.StatusCode != 403 {
		t.Fatalf("expected a 403 StatusError, got: %v", err)
	}
	if fake.calls != 1 {
		t.Errorf("made %d requests, want exactly 1 with no retry across exits", fake.calls)
	}
	for _, proxy := range proxies {
		if m.evictedProxies[proxyKey(&proxy)] {
			t.Errorf("proxy %q was evicted, the download pool must stay untouched", proxy.Host)
		}
	}
}

// TestSetProxyMethodClearsProxyForDirectPages locks in that the main session drops any
// proxy when pages_without_proxy is set, so page requests leave from the IP the clearance
// belongs to even when a single proxy or a loop pool is configured.
func TestSetProxyMethodClearsProxyForDirectPages(t *testing.T) {
	proxies := []http.ProxySettings{
		{Host: "proxy-a", Port: 1080, Enable: true},
		{Host: "proxy-b", Port: 1080, Enable: true},
	}
	m, fake := newProxyTestModule(proxies)
	m.settings.Cloudflare.PagesWithoutProxy = true

	if err := m.setProxyMethod(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if want := []string{""}; len(fake.appliedProxy) != 1 || fake.appliedProxy[0] != want[0] {
		t.Errorf("applied proxies %v, want a single cleared proxy %v", fake.appliedProxy, want)
	}
}
