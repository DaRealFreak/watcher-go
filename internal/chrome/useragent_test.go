package chrome

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
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
// Normalising that token must preserve the browser's native Client Hints too.
func TestSessionUserAgentMatchesBrowser(t *testing.T) {
	testSessionUserAgent(t, "")
}

func TestSessionPreservesUserAgentOverride(t *testing.T) {
	testSessionUserAgent(t, "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 "+
		"(KHTML, like Gecko) Chrome/149.0.0.0 Safari/537.36")
}

func testSessionUserAgent(t *testing.T, configured string) {
	t.Helper()
	if testing.Short() {
		t.Skip("launches a browser")
	}
	if _, err := Find(""); err != nil {
		t.Skipf("no browser available: %v", err)
	}

	// navigator.userAgentData is only populated in a secure context, and
	// http://127.0.0.1 counts as one, so the Client Hints stay readable without
	// reaching out to the network.
	requests := make(chan http.Header, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			select {
			case requests <- r.Header.Clone():
			default:
			}
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html><body>ua probe</body></html>"))
	}))
	defer srv.Close()

	session, err := NewSession(SessionOptions{
		Headless:   true,
		UserAgent:  configured,
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

	// Browser.getVersion provides an independent source for the running binary's
	// identity, unaffected by a page-level user-agent override.
	versionJSON, err := session.call("Browser.getVersion", nil)
	if err != nil {
		t.Fatal(err)
	}
	var version struct {
		Product   string `json:"product"`
		UserAgent string `json:"userAgent"`
	}
	if err = json.Unmarshal(versionJSON, &version); err != nil {
		t.Fatal(err)
	}
	wantAgent := configured
	if wantAgent == "" {
		wantAgent = strings.Replace(version.UserAgent, "HeadlessChrome/", "Chrome/", 1)
	}
	if reported != wantAgent {
		t.Errorf("navigator.userAgent = %q, want %q", reported, wantAgent)
	}

	// Read Client Hints directly: falling back to the UA would conceal missing hints.
	type brandVersion struct {
		Brand   string `json:"brand"`
		Version string `json:"version"`
	}
	var hints struct {
		Brands          []brandVersion `json:"brands"`
		FullVersionList []brandVersion `json:"fullVersionList"`
		Platform        string         `json:"platform"`
		PlatformVersion string         `json:"platformVersion"`
	}
	if err = session.Eval("navigator.userAgentData.getHighEntropyValues(['fullVersionList', 'platformVersion'])", &hints); err != nil {
		t.Fatalf("reading Client Hints: %v", err)
	}
	var major string
	for _, brand := range hints.Brands {
		if brand.Brand == "Chromium" || brand.Brand == "Google Chrome" {
			major = brand.Version
		}
	}
	if major == "" || hints.Platform == "" || hints.PlatformVersion == "" {
		t.Fatalf("browser identity lost its Client Hints: %+v", hints)
	}
	if want, got := chromeMajorVersion(version.Product), major; want != got {
		t.Errorf("binary is Chrome/%s but Client Hints report Chrome/%s", want, got)
	}
	_, fullVersion, _ := strings.Cut(version.Product, "/")
	foundFullVersion := false
	for _, brand := range hints.FullVersionList {
		if (brand.Brand == "Chromium" || brand.Brand == "Google Chrome") && brand.Version == fullVersion {
			foundFullVersion = true
		}
	}
	if !foundFullVersion {
		t.Errorf("Client Hints lack the native full version %q: %+v", fullVersion, hints.FullVersionList)
	}
	if configured == "" && !strings.Contains(reported, "Chrome/"+major+".0.0.0") {
		t.Errorf("expected reduced Chrome version, got %q", reported)
	}

	// The very first target request must carry the same identity as JavaScript.
	select {
	case headers := <-requests:
		if got := headers.Get("User-Agent"); got != userAgent {
			t.Errorf("HTTP User-Agent = %q, want %q", got, userAgent)
		}
		if got := headers.Get("Sec-CH-UA"); !strings.Contains(got, `"Chromium";v="`+major+`"`) {
			t.Errorf("HTTP Client Hints lack Chromium/%s: %q", major, got)
		}
		if got := headers.Get("Sec-CH-UA-Platform"); got != strconv.Quote(hints.Platform) {
			t.Errorf("HTTP platform = %q, JavaScript platform = %q", got, hints.Platform)
		}
	case <-time.After(time.Second):
		t.Fatal("no initial request received")
	}
}
