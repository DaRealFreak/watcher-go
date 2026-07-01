package browserlogin

import (
	"strings"
	"testing"
)

func TestLoginExprEscapesCredentials(t *testing.T) {
	// credentials with quotes/backslashes/newlines must not break the JS expression
	expr := loginExpr(`a"b\c`, "p'w\nd")
	if !strings.Contains(expr, loginFlowFunc) {
		t.Error("expression should embed the login flow function")
	}
	// the raw (unescaped) username must not appear verbatim; it should be JSON-escaped
	if strings.Contains(expr, `a"b\c`) {
		t.Error("username should be JSON-escaped, not embedded raw")
	}
	if !strings.Contains(expr, `"a\"b\\c"`) {
		t.Error("expected JSON-escaped username literal in expression")
	}
}
