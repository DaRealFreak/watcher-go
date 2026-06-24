package fourchan

import (
	"errors"
	"testing"

	"github.com/DaRealFreak/watcher-go/internal/http"
	"github.com/DaRealFreak/watcher-go/internal/http/tls_session"
	fhttp "github.com/bogdanfinn/fhttp"
)

// getResult scripts one fakeSession.Get outcome.
type getResult struct {
	statusCode int
	err        error
}

// fakeSession is a minimal TlsClientSessionInterface used to drive getPage without
// performing real network requests. Only Get and SetProxy are exercised; any other
// method panics (via the embedded nil interface) to surface unexpected usage.
type fakeSession struct {
	http.TlsClientSessionInterface
	results      []getResult
	calls        int
	appliedProxy []string
}

func (f *fakeSession) Get(_ string, _ ...http.TlsClientErrorHandler) (*fhttp.Response, error) {
	result := f.results[f.calls]
	f.calls++

	return &fhttp.Response{StatusCode: result.statusCode}, result.err
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
