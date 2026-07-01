package chrome

import (
	"encoding/json"
	"fmt"
	"time"

	fhttp "github.com/bogdanfinn/fhttp"
	"github.com/gorilla/websocket"
)

// DefaultUserAgent is a current desktop Chrome UA used for browser sessions.
const DefaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) " +
	"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Safari/537.36"

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
	inst    *Instance
	conn    *websocket.Conn
	id      int
	timeout time.Duration
}

// SessionOptions configures a browser session.
type SessionOptions struct {
	// BrowserPath is the Chrome/Chromium executable. Empty auto-detects an
	// installed browser and, failing that, downloads Chrome for Testing.
	BrowserPath string
	// Headless runs Chrome without a window (default true; set false to debug).
	Headless bool
	// UserAgent overrides the browser user agent (defaults to DefaultUserAgent).
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
	if opts.UserAgent == "" {
		opts.UserAgent = DefaultUserAgent
	}

	execPath, err := Find(opts.BrowserPath)
	if err != nil {
		return nil, err
	}

	inst, err := Launch(LaunchOptions{
		ExecPath:   execPath,
		Headless:   opts.Headless,
		UserAgent:  opts.UserAgent,
		InitialURL: opts.InitialURL,
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
	return s, nil
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
