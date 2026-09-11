package browserlogin

import (
	"strings"
	"testing"
)

func TestFillExprsEscapeCredentials(t *testing.T) {
	// credentials with quotes/backslashes/newlines must not break the JS expression
	cases := map[string]string{
		"username": fillUsernameExpr(`a"b\c`),
		"password": fillPasswordExpr(`a"b\c`),
	}

	for field, expr := range cases {
		// the raw (unescaped) credential must not appear verbatim
		if strings.Contains(expr, `a"b\c`) {
			t.Errorf("%s should be JSON-escaped, not embedded raw:\n%s", field, expr)
		}
		if !strings.Contains(expr, `"a\"b\\c"`) {
			t.Errorf("expected JSON-escaped %s literal in expression:\n%s", field, expr)
		}
	}
}

func TestFillExprsTargetTheRightForm(t *testing.T) {
	username := fillUsernameExpr("someone")
	if !strings.Contains(username, "form[action='/_sisu/do/step2']") {
		t.Errorf("username step should submit the step2 form:\n%s", username)
	}
	if !strings.Contains(username, "input[name='username']") {
		t.Errorf("username step should fill the username field:\n%s", username)
	}

	password := fillPasswordExpr("secret")
	if !strings.Contains(password, "form[action='/_sisu/do/signin']") {
		t.Errorf("password step should submit the signin form:\n%s", password)
	}
	if !strings.Contains(password, "input[name='password']") {
		t.Errorf("password step should fill the password field:\n%s", password)
	}

	// no unresolved format verbs left in either expression
	for _, expr := range []string{username, password} {
		if strings.Contains(expr, "%!") || strings.Contains(expr, "%[") {
			t.Errorf("expression has unresolved format verbs:\n%s", expr)
		}
	}
}

// TestSubmitsViaFormNavigation guards the reason the login works at all:
// PerimeterX blocks the signin endpoint when the credentials are posted with
// fetch() instead of a real top-level form navigation.
func TestSubmitsViaFormNavigation(t *testing.T) {
	for _, expr := range []string{fillUsernameExpr("someone"), fillPasswordExpr("secret")} {
		if !strings.Contains(expr, "form.submit()") {
			t.Errorf("login step must submit the form as a navigation:\n%s", expr)
		}
		if strings.Contains(expr, "fetch(") {
			t.Errorf("login step must not post credentials with fetch():\n%s", expr)
		}
	}
}
