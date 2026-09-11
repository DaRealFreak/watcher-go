// Package browserlogin performs the DeviantArt login inside a real Chrome
// instance, using the undetected CDP session from internal/chrome.
//
// DeviantArt guards its /_sisu login endpoints with PerimeterX bot detection.
// Getting through it requires (1) running the PerimeterX sensor JavaScript in a
// real browser to mint the PerimeterX cookies, (2) not tripping PerimeterX's
// CDP-automation detection on the signin endpoint, and (3) submitting the login
// forms the way the browser itself would. chrome.Session provides the first two
// (a real browser that never calls Runtime.enable and reports a user agent
// matching its own build); this package only encodes the DeviantArt-specific
// login flow and hands the harvested session cookies back to the fast TLS HTTP
// client.
package browserlogin

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/DaRealFreak/watcher-go/internal/chrome"
	http "github.com/bogdanfinn/fhttp"
)

const (
	loginURL     = "https://www.deviantart.com/users/login"
	cookieOrigin = "https://www.deviantart.com"
	// stepTimeout bounds how long a single login step may take to settle.
	stepTimeout = 45 * time.Second
)

// Options configures the browser login.
type Options struct {
	// ChromePath is the Chrome/Chromium executable. Empty auto-detects an
	// installed browser (and downloads Chrome for Testing if none is found).
	ChromePath string
	// Headless runs Chrome without a visible window (default true; set false to debug).
	Headless bool
	// UserAgent overrides the browser user agent. Leave empty (recommended) so
	// Chrome reports its own; a hardcoded UA drifts out of sync with the
	// installed browser on every Chrome update, and PerimeterX blocks the
	// signin endpoint when the UA contradicts the Client Hints.
	UserAgent string
	// Timeout bounds the whole login (defaults to 90s).
	Timeout time.Duration
	// SensorWait is how long to let the PerimeterX sensor run before logging in.
	SensorWait time.Duration
}

// pageState is a snapshot of the login page as seen from inside the browser.
type pageState struct {
	URL         string `json:"url"`
	Blocked     bool   `json:"blocked"`
	HasPassword bool   `json:"hasPassword"`
	LoggedInAs  string `json:"loggedInAs"`
	Error       string `json:"error"`
	Text        string `json:"text"`
}

// usernameRejected reports whether the step2 endpoint answered without a
// password form, which is how DeviantArt refuses an unknown username. The page
// only reaches that URL once the submission navigated, so this cannot match the
// form we just submitted.
func (s pageState) usernameRejected() bool {
	return strings.Contains(s.URL, "/_sisu/do/step2") && !s.HasPassword
}

// passwordRejected reports whether the signin endpoint answered by re-rendering
// the password form, which is how DeviantArt refuses credentials.
func (s pageState) passwordRejected() bool {
	return strings.Contains(s.URL, "/_sisu/do/signin") && s.HasPassword
}

// describe returns the most specific explanation the page offers.
func (s pageState) describe() string {
	if s.Error != "" {
		return s.Error
	}
	return summarise(s.Text)
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

	// let the PerimeterX sensor run and mint its cookies before logging in
	time.Sleep(opts.SensorWait)

	// step 1: submit the username and wait for the password step
	if err = submit(session, fillUsernameExpr(username)); err != nil {
		return nil, fmt.Errorf("browserlogin: submitting the username: %w", err)
	}

	state, err := waitForState(session, func(s pageState) bool {
		return s.HasPassword || s.usernameRejected()
	})
	if err != nil {
		return nil, fmt.Errorf("browserlogin: waiting for the password step: %w", err)
	}
	if err = checkState(state, "username"); err != nil {
		return nil, err
	}
	if !state.HasPassword {
		return nil, fmt.Errorf("browserlogin: username was not accepted: %s", state.describe())
	}

	// step 2: submit the password and wait for the session to be established
	if err = submit(session, fillPasswordExpr(password)); err != nil {
		return nil, fmt.Errorf("browserlogin: submitting the password: %w", err)
	}

	state, err = waitForState(session, func(s pageState) bool {
		return s.LoggedInAs != "" || s.passwordRejected()
	})
	if err != nil {
		return nil, fmt.Errorf("browserlogin: waiting for the signin result: %w", err)
	}
	if err = checkState(state, "signin"); err != nil {
		return nil, err
	}
	if state.LoggedInAs == "" {
		return nil, fmt.Errorf(
			"browserlogin: login not confirmed (wrong credentials, 2FA, or captcha): %s",
			state.describe())
	}

	cookies, err := session.Cookies(cookieOrigin)
	if err != nil {
		return nil, fmt.Errorf("browserlogin: harvesting cookies: %w", err)
	}

	slog.Debug(fmt.Sprintf("browserlogin: authenticated as %q, harvested %d cookies",
		state.LoggedInAs, len(cookies)))

	return cookies, nil
}

// checkState turns a PerimeterX block page into an error naming the stage it hit.
func checkState(state pageState, stage string) error {
	if state.Blocked {
		return fmt.Errorf("browserlogin: PerimeterX blocked the %s step (automation detected)", stage)
	}
	return nil
}

// submit runs an in-page expression that fills and submits a login form, and
// reports the failure the page describes when the form is not there.
func submit(session *chrome.Session, expression string) error {
	var result string
	if err := session.Eval(expression, &result); err != nil {
		return err
	}
	if result != "" {
		return fmt.Errorf("%s", result)
	}
	return nil
}

// waitForState polls the page until done reports the expected state, the page is
// blocked, or the step times out. Submitting a form navigates the page, so the
// result has to be polled rather than awaited.
func waitForState(session *chrome.Session, done func(pageState) bool) (pageState, error) {
	var (
		state    pageState
		deadline = time.Now().Add(stepTimeout)
	)

	for time.Now().Before(deadline) {
		// the evaluation fails while the document is being swapped out
		if err := session.Eval(pageStateExpr, &state); err == nil {
			if state.Blocked || done(state) {
				return state, nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	return state, fmt.Errorf("timed out after %s (at %s)", stepTimeout, state.URL)
}

// summarise trims page text down to something printable in an error message.
func summarise(text string) string {
	text = strings.Join(strings.Fields(text), " ")
	if len(text) > 200 {
		text = text[:200] + "..."
	}
	if text == "" {
		return "(no page text)"
	}
	return text
}

// fillUsernameExpr fills the username into the login form and submits it.
func fillUsernameExpr(username string) string {
	return fmt.Sprintf(submitFormFunc, "/_sisu/do/step2", "username", jsonString(username))
}

// fillPasswordExpr fills the password into the signin form and submits it.
func fillPasswordExpr(password string) string {
	return fmt.Sprintf(submitFormFunc, "/_sisu/do/signin", "password", jsonString(password))
}

// jsonString renders a Go string as a JavaScript string literal, so credentials
// containing quotes or backslashes cannot break out of the expression.
func jsonString(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

// submitFormFunc fills one field of a DeviantArt login form and submits the form
// itself.
//
// The form is submitted through HTMLFormElement.submit(), which performs a real
// top-level navigation. PerimeterX rejects the same credentials when they are
// posted with fetch(): a background request carries Sec-Fetch-Mode: cors and
// Sec-Fetch-Dest: empty instead of the navigate/document pair a submitted form
// produces, and the signin endpoint is enforced on exactly that difference.
const submitFormFunc = `(() => {
  const form = document.querySelector("form[action='%[1]s']");
  if (!form) return 'no form[action=%[1]s] on ' + location.href;
  const field = form.querySelector("input[name='%[2]s']");
  if (!field) return 'no %[2]s field on ' + location.href;
  field.value = %[3]s;
  const remember = form.querySelector("input[name='remember']");
  if (remember) remember.checked = true;
  form.submit();
  return '';
})()`

// pageStateExpr reports where the login flow currently stands. The userinfo
// cookie only carries a username once the session is authenticated.
const pageStateExpr = `(() => {
  const match = decodeURIComponent(document.cookie).match(/username":"([^"]*)"/);
  // DeviantArt renders the reason as plain text next to the field it belongs to
  const error = [...document.querySelectorAll('p, span, div, label')]
    .map(el => (el.textContent || '').trim())
    .filter(text => text && text.length < 200 &&
      /incorrect|invalid|does not match|try again|too many attempts|locked|suspended/i.test(text))
    // the innermost element holding the message is the shortest match
    .sort((a, b) => a.length - b.length)[0] || '';
  return {
    url: location.href,
    blocked: document.documentElement.innerHTML.includes('Access to this page has been denied'),
    hasPassword: !!document.querySelector("input[name='password']"),
    loggedInAs: match ? match[1] : '',
    error: error,
    text: (document.body ? document.body.innerText || '' : '').trim().slice(0, 400),
  };
})()`
