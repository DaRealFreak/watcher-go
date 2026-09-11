package chrome

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestChromeMajorVersion(t *testing.T) {
	cases := map[string]string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/152.0.0.0 Safari/537.36":         "152",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/152.0.0.0 Safari/537.36": "152",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:139.0) Gecko/20100101 Firefox/139.0":                                        "",
		"": "",
	}
	for userAgent, want := range cases {
		if got := chromeMajorVersion(userAgent); got != want {
			t.Errorf("chromeMajorVersion(%q) = %q, want %q", userAgent, got, want)
		}
	}
}

// TestSessionUserAgentMatchesBrowser is the regression test for the DeviantArt
// login breaking after a Chrome update: the session used to report a hardcoded
// Chrome version, and headless Chrome reports "HeadlessChrome" when left alone.
// Both contradict the Client Hints (which no --user-agent switch can change),
// and PerimeterX rejects the login on exactly that contradiction.
func TestSessionUserAgentMatchesBrowser(t *testing.T) {
	if testing.Short() {
		t.Skip("launches a browser")
	}
	if _, err := Find(""); err != nil {
		t.Skipf("no browser available: %v", err)
	}

	// navigator.userAgentData is only populated in a secure context, and
	// http://127.0.0.1 counts as one, so the Client Hints stay readable without
	// reaching out to the network.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html><body>ua probe</body></html>"))
	}))
	defer srv.Close()

	session, err := NewSession(SessionOptions{
		Headless:   true,
		InitialURL: srv.URL,
		Timeout:    60 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	defer func() { _ = session.Close() }()

	userAgent := session.UserAgent()
	if userAgent == "" {
		t.Fatal("session reported no user agent")
	}

	// an explicit automation marker is what PerimeterX keys on
	if strings.Contains(userAgent, "Headless") {
		t.Errorf("user agent advertises headless automation: %q", userAgent)
	}

	// the page must report the same string the session claims
	var reported string
	if err = session.Eval("navigator.userAgent", &reported); err != nil {
		t.Fatalf("reading navigator.userAgent: %v", err)
	}
	if reported != userAgent {
		t.Errorf("navigator.userAgent = %q, session reports %q", reported, userAgent)
	}

	// and it must agree with the Client Hints, which keep reporting the real build
	if want, got := chromeMajorVersion(reported), browserMajorVersion(session); want != got {
		t.Errorf("user agent claims Chrome/%s but Client Hints report Chrome/%s (%q)",
			want, got, reported)
	}
}
