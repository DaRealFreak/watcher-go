package chrome

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	fhttp "github.com/bogdanfinn/fhttp"
	"github.com/gorilla/websocket"
)

// chromeVersionPattern extracts the Chrome major version from a user agent.
var chromeVersionPattern = regexp.MustCompile(`Chrome/(\d+)`)

// Session is a real Chrome instance driven over the DevTools Protocol in a way
// that stays invisible to CDP-based bot detection (PerimeterX, Cloudflare,
// DataDome, ...).
//
// The key property is that it NEVER calls Runtime.enable (or any other *.enable
// domain). Enabling the Runtime domain is the classic signal anti-bot vendors
// use to detect automated browsers, so this driver issues only plain commands
// (Runtime.evaluate against the page's default context, Page.navigate,
// Network.getCookies). Chrome is also launched without --enable-automation, so
// navigator.webdriver stays false.
//
// Use it to run a real browser through a JavaScript challenge or login, then
// hand the resulting cookies to a lightweight HTTP client for the actual work.
// See the package README for end-to-end patterns.
type Session struct {
	inst      *Instance
	conn      *websocket.Conn
	id        int
	timeout   time.Duration
	userAgent string
}

// SessionOptions configures a browser session.
type SessionOptions struct {
	// BrowserPath is the Chrome/Chromium executable. Empty auto-detects an
	// installed browser and, failing that, downloads Chrome for Testing.
	BrowserPath string
	// Headless runs Chrome without a window (default true; set false to debug).
	Headless bool
	// UserAgent overrides the browser user agent. Leave empty (recommended) to
	// use Chrome's own user agent.
	//
	// Chrome's --user-agent switch only rewrites navigator.userAgent and the
	// User-Agent header; it does NOT update the Client Hints (navigator.
	// userAgentData / Sec-CH-UA), which keep reporting the real build. A spoofed
	// UA whose version disagrees with the browser is therefore trivially
	// detectable and is treated as automation by bot protection such as
	// PerimeterX, so only override this when the value tracks the actual browser.
	UserAgent string
	// InitialURL is opened on launch (defaults to about:blank).
	InitialURL string
	// Timeout bounds each CDP command and is the default for the Wait* helpers.
	Timeout time.Duration
}

// NewSession locates/launches an undetected Chrome and connects to its page
// target. Call Close when done.
func NewSession(opts SessionOptions) (*Session, error) {
	if opts.Timeout == 0 {
		opts.Timeout = 90 * time.Second
	}

	execPath, err := Find(opts.BrowserPath)
	if err != nil {
		return nil, err
	}

	// Chrome's local version page is a secure context, so it exposes native
	// Client Hints without sending a request to the target site. about:blank
	// does not expose navigator.userAgentData.
	inst, err := Launch(LaunchOptions{
		ExecPath:   execPath,
		Headless:   opts.Headless,
		InitialURL: "chrome://version",
	})
	if err != nil {
		return nil, err
	}

	conn, _, err := websocket.DefaultDialer.Dial(inst.PageWSURL, nil)
	if err != nil {
		_ = inst.Close()
		return nil, fmt.Errorf("connecting to chrome devtools: %w", err)
	}

	s := &Session{inst: inst, conn: conn, timeout: opts.Timeout}
	s.WaitReady(opts.Timeout)

	if err = s.applyUserAgent(opts.UserAgent); err != nil {
		_ = s.Close()
		return nil, fmt.Errorf("configuring chrome user agent: %w", err)
	}
	slog.Debug(fmt.Sprintf("chrome: using browser user agent: %s", s.userAgent))

	initialURL := opts.InitialURL
	if initialURL == "" {
		initialURL = "about:blank"
	}
	if err = s.Navigate(initialURL); err != nil {
		_ = s.Close()
		return nil, fmt.Errorf("opening %s: %w", initialURL, err)
	}

	return s, nil
}

// applyUserAgent makes the browser report a user agent that matches the build it
// actually runs.
//
// Headless Chrome advertises itself as "HeadlessChrome/<version>", which is an
// explicit automation marker, while a hardcoded override drifts out of sync with
// the installed browser on every Chrome update. Either way the user agent ends up
// contradicting the Client Hints. Deriving both from the running browser keeps
// them in agreement across updates.
func (s *Session) applyUserAgent(configured string) error {
	var actual string
	if err := s.Eval("navigator.userAgent", &actual); err != nil {
		return err
	}
	s.userAgent = actual

	wanted := configured
	if wanted == "" {
		wanted = strings.Replace(actual, "HeadlessChrome/", "Chrome/", 1)
	} else {
		s.warnOnUserAgentMismatch(configured)
	}

	if wanted == "" || wanted == actual {
		return nil
	}

	// CDP suppresses Client Hints when a UA override omits userAgentMetadata.
	// Preserve all native values, including high-entropy version and OS details,
	// instead of inventing a second browser identity to maintain across updates.
	var metadata map[string]any
	if err := s.Eval(`navigator.userAgentData.getHighEntropyValues([
      'architecture', 'bitness', 'model', 'platformVersion',
      'fullVersionList', 'wow64', 'formFactors'
    ])`, &metadata); err != nil {
		return fmt.Errorf("reading native Client Hints: %w", err)
	}
	if len(metadata) == 0 {
		return fmt.Errorf("browser returned no native Client Hints")
	}

	// This is a plain command; no CDP domain needs to be enabled.
	if _, err := s.call("Emulation.setUserAgentOverride", map[string]any{
		"userAgent":         wanted,
		"userAgentMetadata": metadata,
	}); err != nil {
		return err
	}
	s.userAgent = wanted

	return nil
}

// UserAgent returns the user agent the browser actually sends. Use it to align
// an HTTP client that reuses this session's cookies with the browser that
// minted them.
func (s *Session) UserAgent() string {
	return s.userAgent
}

// warnOnUserAgentMismatch logs when a configured user agent claims a different
// Chrome major version than the running browser. Client Hints still report the
// real version, so the contradiction marks the session as automated.
func (s *Session) warnOnUserAgentMismatch(configured string) {
	if configured == "" || s.userAgent == "" {
		return
	}

	want, got := chromeMajorVersion(configured), browserMajorVersion(s)
	if want == "" || got == "" || want == got {
		return
	}

	slog.Warn(fmt.Sprintf(
		"chrome: configured user agent reports Chrome/%s but the browser is Chrome/%s; "+
			"Client Hints still report %s, and bot protection treats that mismatch as automation. "+
			"Remove the override to use the browser's own user agent.", want, got, got))
}

// browserMajorVersion reports the Chrome major version the browser advertises
// through Client Hints (which a --user-agent override cannot change), falling
// back to the reported user agent.
func browserMajorVersion(s *Session) string {
	var version string
	// navigator.userAgentData is immune to the --user-agent switch
	if err := s.Eval(`(() => {
      const d = navigator.userAgentData;
      if (!d || !d.brands) return '';
      const b = d.brands.find(b => /Chrome|Chromium/i.test(b.brand) && !/Not/i.test(b.brand));
      return b ? String(b.version) : '';
    })()`, &version); err == nil && version != "" {
		return version
	}
	return chromeMajorVersion(s.userAgent)
}

// chromeMajorVersion extracts the Chrome major version from a user agent,
// returning "" when the string carries no Chrome version.
func chromeMajorVersion(userAgent string) string {
	if m := chromeVersionPattern.FindStringSubmatch(userAgent); m != nil {
		return m[1]
	}
	return ""
}

// Close terminates the browser and releases its resources.
func (s *Session) Close() error {
	if s.conn != nil {
		_ = s.conn.Close()
	}
	if s.inst != nil {
		return s.inst.Close()
	}
	return nil
}

// Eval runs a JavaScript expression in the page's default execution context and
// unmarshals its (by-value) result into out. Promises are awaited, so async
// arrow functions work. It never calls Runtime.enable.
func (s *Session) Eval(expression string, out any) error {
	res, err := s.call("Runtime.evaluate", map[string]any{
		"expression":    expression,
		"awaitPromise":  true,
		"returnByValue": true,
	})
	if err != nil {
		return err
	}

	var r struct {
		Result struct {
			Value json.RawMessage `json:"value"`
		} `json:"result"`
		ExceptionDetails *struct {
			Text string `json:"text"`
		} `json:"exceptionDetails"`
	}
	if err = json.Unmarshal(res, &r); err != nil {
		return err
	}
	if r.ExceptionDetails != nil {
		return fmt.Errorf("evaluate raised an exception: %s", r.ExceptionDetails.Text)
	}
	if out != nil && len(r.Result.Value) > 0 {
		return json.Unmarshal(r.Result.Value, out)
	}
	return nil
}

// Navigate loads url (via Page.navigate, without Page.enable) and waits for the
// document to finish loading.
func (s *Session) Navigate(url string) error {
	if _, err := s.call("Page.navigate", map[string]any{"url": url}); err != nil {
		return err
	}
	s.WaitReady(s.timeout)
	return nil
}

// Cookies returns the browser's cookies for url (including HttpOnly cookies,
// which document.cookie cannot see), mapped to fhttp cookies for reuse by the
// TLS HTTP client.
func (s *Session) Cookies(url string) ([]*fhttp.Cookie, error) {
	res, err := s.call("Network.getCookies", map[string]any{"urls": []string{url}})
	if err != nil {
		return nil, err
	}

	var r struct {
		Cookies []cdpCookie `json:"cookies"`
	}
	if err = json.Unmarshal(res, &r); err != nil {
		return nil, err
	}
	return mapCookies(r.Cookies), nil
}

// WaitReady polls document.readyState until it reports "complete" or timeout.
func (s *Session) WaitReady(timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var rs string
		if err := s.Eval("document.readyState", &rs); err == nil && rs == "complete" {
			return
		}
		time.Sleep(300 * time.Millisecond)
	}
}

// WaitFor polls a boolean JavaScript expression until it evaluates to true or
// the timeout elapses. Useful for waiting on a challenge to clear.
func (s *Session) WaitFor(boolExpr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var ok bool
		if err := s.Eval(boolExpr, &ok); err == nil && ok {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for %q", boolExpr)
}

// WaitForCookie polls until a cookie with the given name exists for url (e.g.
// Cloudflare's cf_clearance once the challenge is solved) or timeout elapses.
// It uses Network.getCookies, so it also sees HttpOnly cookies.
func (s *Session) WaitForCookie(name, url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		cookies, err := s.Cookies(url)
		if err == nil {
			for _, c := range cookies {
				if c.Name == name && c.Value != "" {
					return nil
				}
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for cookie %q", name)
}

// call sends a CDP command and returns its result, skipping unsolicited events.
func (s *Session) call(method string, params map[string]any) (json.RawMessage, error) {
	s.id++
	id := s.id

	msg := map[string]any{"id": id, "method": method}
	if params != nil {
		msg["params"] = params
	}

	_ = s.conn.SetWriteDeadline(time.Now().Add(s.timeout))
	if err := s.conn.WriteJSON(msg); err != nil {
		return nil, err
	}

	deadline := time.Now().Add(s.timeout)
	for {
		_ = s.conn.SetReadDeadline(deadline)
		var resp struct {
			ID     int             `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := s.conn.ReadJSON(&resp); err != nil {
			return nil, err
		}
		if resp.ID != id {
			continue // unsolicited event or a different response
		}
		if resp.Error != nil {
			return nil, fmt.Errorf("cdp %s: %s", method, resp.Error.Message)
		}
		return resp.Result, nil
	}
}

// cdpCookie is a cookie as returned by Network.getCookies.
type cdpCookie struct {
	Name     string  `json:"name"`
	Value    string  `json:"value"`
	Domain   string  `json:"domain"`
	Path     string  `json:"path"`
	Expires  float64 `json:"expires"`
	HTTPOnly bool    `json:"httpOnly"`
	Secure   bool    `json:"secure"`
}

// mapCookies converts CDP cookies into fhttp cookies usable by the TLS client.
func mapCookies(in []cdpCookie) []*fhttp.Cookie {
	out := make([]*fhttp.Cookie, 0, len(in))
	for _, c := range in {
		cookie := &fhttp.Cookie{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Secure:   c.Secure,
			HttpOnly: c.HTTPOnly,
		}
		// Expires is unix seconds; <= 0 means a session cookie.
		if c.Expires > 0 {
			cookie.Expires = time.Unix(int64(c.Expires), 0)
		}
		out = append(out, cookie)
	}
	return out
}
