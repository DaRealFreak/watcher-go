package login

import (
	"io"
	"strings"
	"testing"

	http "github.com/bogdanfinn/fhttp"
)

// response builds a minimal *http.Response wrapping the given HTML body.
func response(html string) *http.Response {
	return &http.Response{Body: io.NopCloser(strings.NewReader(html))}
}

// TestGetLoginCSRFToken locks the parsing contract that both the credential
// login and the IsLoggedIn session check depend on: the username step yields
// csrf+lu_token, the password step additionally yields lu_token2, and a
// logged-in page yields a csrf token via the script fallback with no lu_token.
func TestGetLoginCSRFToken(t *testing.T) {
	g := DeviantArtLogin{}

	t.Run("login page (logged out)", func(t *testing.T) {
		html := `<html><body>
			<form action="/_sisu/do/step2">
				<input name="csrf_token" value="CSRF123"/>
				<input name="lu_token" value="LU123"/>
				<input name="username"/>
			</form>
		</body></html>`

		info, err := g.GetLoginCSRFToken(response(html))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if info.CSRFToken != "CSRF123" {
			t.Errorf("CSRFToken = %q, want CSRF123", info.CSRFToken)
		}
		if info.LuToken != "LU123" {
			t.Errorf("LuToken = %q, want LU123", info.LuToken)
		}
		if info.LuToken2 != "" {
			t.Errorf("LuToken2 = %q, want empty on the username step", info.LuToken2)
		}
	})

	t.Run("password step (signin form)", func(t *testing.T) {
		html := `<html><body>
			<form action="/_sisu/do/signin">
				<input name="csrf_token" value="CSRF456"/>
				<input name="lu_token" value="LU456"/>
				<input name="lu_token2" value="UUID-789"/>
				<input name="password"/>
			</form>
		</body></html>`

		info, err := g.GetLoginCSRFToken(response(html))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if info.CSRFToken != "CSRF456" || info.LuToken != "LU456" || info.LuToken2 != "UUID-789" {
			t.Errorf("unexpected info on password step: %+v", info)
		}
	})

	t.Run("logged-in page (csrf via script, no lu_token)", func(t *testing.T) {
		html := `<html><head>
			<script>window.__CSRF_TOKEN__ = 'CSRFXYZ';</script>
		</head><body>welcome back</body></html>`

		info, err := g.GetLoginCSRFToken(response(html))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// this is exactly the IsLoggedIn signal: csrf present, lu_token absent
		if info.CSRFToken != "CSRFXYZ" {
			t.Errorf("CSRFToken = %q, want CSRFXYZ (script fallback)", info.CSRFToken)
		}
		if info.LuToken != "" || info.LuToken2 != "" {
			t.Errorf("expected no lu_tokens on a logged-in page: %+v", info)
		}
	})

	t.Run("no csrf token anywhere is an error", func(t *testing.T) {
		if _, err := g.GetLoginCSRFToken(response(`<html><body>nope</body></html>`)); err == nil {
			t.Error("expected an error when no csrf token can be found")
		}
	})
}
