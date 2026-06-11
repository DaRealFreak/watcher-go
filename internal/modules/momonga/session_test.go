package momonga

import (
	"strings"
	"testing"

	http "github.com/bogdanfinn/fhttp"
)

// TestApplyHeadersSetsBrowserUserAgent locks in the fix for momon-ga.com's 403 on
// requests without a recognized browser User-Agent. Pre-fix code set no User-Agent.
func TestApplyHeadersSetsBrowserUserAgent(t *testing.T) {
	m := newTestModule()

	req, err := http.NewRequest("GET", "https://momon-ga.com/cartoonist/zunta/", nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}

	m.applyHeaders(req)

	if ua := req.Header.Get("User-Agent"); !strings.Contains(ua, "Firefox") {
		t.Errorf("expected a browser User-Agent header, got %q", ua)
	}
	if req.Header.Get("Accept") == "" {
		t.Error("expected an Accept header to be set")
	}
}
