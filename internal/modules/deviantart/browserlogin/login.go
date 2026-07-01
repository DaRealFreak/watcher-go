// Package browserlogin performs the DeviantArt login inside a real Chrome
// instance, using the undetected CDP session from internal/chrome.
//
// DeviantArt guards its /_sisu login endpoints with PerimeterX bot detection.
// Getting through it requires (1) running the PerimeterX sensor JavaScript in a
// real browser to mint a valid _px cookie, and (2) not tripping PerimeterX's
// CDP-automation detection on the signin endpoint. chrome.Session provides both
// (a real browser that never calls Runtime.enable); this package only encodes
// the DeviantArt-specific login flow and hands the harvested session cookies
// back to the fast TLS HTTP client.
package browserlogin

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/DaRealFreak/watcher-go/internal/chrome"
	http "github.com/bogdanfinn/fhttp"
)

const (
	loginURL     = "https://www.deviantart.com/users/login"
	cookieOrigin = "https://www.deviantart.com"
)

// Options configures the browser login.
type Options struct {
	// ChromePath is the Chrome/Chromium executable. Empty auto-detects an
	// installed browser (and downloads Chrome for Testing if none is found).
	ChromePath string
	// Headless runs Chrome without a visible window (default true; set false to debug).
	Headless bool
	// UserAgent overrides the browser user agent (defaults to chrome.DefaultUserAgent).
	UserAgent string
	// Timeout bounds the whole login (defaults to 90s).
	Timeout time.Duration
	// SensorWait is how long to let the PerimeterX sensor run before logging in.
	SensorWait time.Duration
}

// loginResult mirrors the object returned by the in-page login flow.
type loginResult struct {
	OK       bool   `json:"ok"`
	Stage    string `json:"stage"`
	Status   int    `json:"status"`
	Username string `json:"username"`
	Msg      string `json:"msg"`
}

// Login drives Chrome through the PerimeterX-protected login and returns the
// resulting DeviantArt session cookies on success.
func Login(username, password string, opts Options) ([]*http.Cookie, error) {
	if opts.SensorWait == 0 {
		opts.SensorWait = 5 * time.Second
	}

	session, err := chrome.NewSession(chrome.SessionOptions{
		BrowserPath: opts.ChromePath,
		Headless:    opts.Headless,
		UserAgent:   opts.UserAgent,
		InitialURL:  loginURL,
		Timeout:     opts.Timeout,
	})
	if err != nil {
		return nil, fmt.Errorf("browserlogin: %w", err)
	}
	defer func() { _ = session.Close() }()

	// let the PerimeterX sensor run and mint the _px cookie before logging in
	time.Sleep(opts.SensorWait)

	var result loginResult
	if err = session.Eval(loginExpr(username, password), &result); err != nil {
		return nil, fmt.Errorf("browserlogin: login flow failed: %w", err)
	}

	if !result.OK {
		msg := result.Msg
		if msg == "" {
			msg = "login not confirmed"
		}
		return nil, fmt.Errorf("browserlogin: login failed at stage %q (status %d): %s",
			result.Stage, result.Status, msg)
	}

	cookies, err := session.Cookies(cookieOrigin)
	if err != nil {
		return nil, fmt.Errorf("browserlogin: harvesting cookies: %w", err)
	}

	slog.Debug(fmt.Sprintf("browserlogin: authenticated as %q, harvested %d cookies",
		result.Username, len(cookies)))
	return cookies, nil
}

// loginExpr builds the in-page async login flow, injecting the credentials as
// JSON string literals so any characters are escaped safely.
func loginExpr(username, password string) string {
	u, _ := json.Marshal(username)
	p, _ := json.Marshal(password)
	return fmt.Sprintf("(%s)(%s, %s)", loginFlowFunc, string(u), string(p))
}

// loginFlowFunc replicates the DeviantArt login (login page -> step2 -> signin)
// using same-origin fetch, then confirms authentication via the userinfo cookie
// (its username is only populated once logged in).
const loginFlowFunc = `async (username, password) => {
  const parse = (html, sel) => {
    const doc = new DOMParser().parseFromString(html, 'text/html');
    const el = doc.querySelector(sel);
    return el ? el.getAttribute('value') : null;
  };
  const form = (html) => ({
    csrf: parse(html, "form[action='/_sisu/do/step2'] input[name='csrf_token'], form[action='/_sisu/do/signin'] input[name='csrf_token']"),
    lu:   parse(html, "form[action='/_sisu/do/step2'] input[name='lu_token'], form[action='/_sisu/do/signin'] input[name='lu_token']"),
    lu2:  parse(html, "form[action='/_sisu/do/step2'] input[name='lu_token2'], form[action='/_sisu/do/signin'] input[name='lu_token2']"),
  });
  const post = (url, data) => fetch(url, {
    method: 'POST', credentials: 'include',
    headers: {'Content-Type': 'application/x-www-form-urlencoded'},
    body: new URLSearchParams(data).toString()
  });
  const loggedInName = () => {
    try {
      const m = decodeURIComponent(document.cookie).match(/username":"([^"]*)"/);
      return m ? m[1] : '';
    } catch (e) { return ''; }
  };

  // step 1: login page -> csrf + lu_token
  let f = form(await (await fetch("https://www.deviantart.com/users/login", {credentials:'include'})).text());
  if (!f.csrf || !f.lu) return {ok:false, stage:'login-page', status:0, msg:'could not read csrf/lu_token from login page'};

  // step 2: submit username -> password step with lu_token2
  const r2 = await post("https://www.deviantart.com/_sisu/do/step2", {
    referer:"https://www.deviantart.com/", referer_type:"", csrf_token:f.csrf,
    challenge:"0", lu_token:f.lu, username, remember:"on"
  });
  f = form(await r2.text());
  if (!f.lu2) return {ok:false, stage:'step2', status:r2.status, msg:'username rejected or PerimeterX blocked step2'};

  // step 3: submit password
  const r3 = await post("https://www.deviantart.com/_sisu/do/signin", {
    referer:"https://www.deviantart.com/_sisu/do/step2", referer_type:"", csrf_token:f.csrf,
    challenge:"0", lu_token:f.lu, lu_token2:f.lu2, username:"", password, remember:"on"
  });
  const body = await r3.text();
  if (body.includes('Access to this page has been denied')) {
    return {ok:false, stage:'signin', status:r3.status, msg:'PerimeterX blocked signin (automation detected)'};
  }

  // confirm: the userinfo cookie carries the username only once authenticated
  const name = loggedInName();
  return {ok: name !== '', stage: name !== '' ? 'done' : 'signin', status:r3.status,
          username: name, msg: name !== '' ? '' : 'login not confirmed (wrong credentials, 2FA, or captcha)'};
}`
