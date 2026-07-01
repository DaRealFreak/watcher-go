package chrome

import "testing"

func TestMapCookies(t *testing.T) {
	in := []cdpCookie{
		{Name: "auth", Value: "abc", Domain: ".deviantart.com", Path: "/", Secure: true, HTTPOnly: true, Expires: 1893456000},
		{Name: "session", Value: "xyz", Domain: "www.deviantart.com", Path: "/", Expires: -1}, // session cookie
	}
	out := mapCookies(in)

	if len(out) != 2 {
		t.Fatalf("expected 2 cookies, got %d", len(out))
	}

	auth := out[0]
	if auth.Name != "auth" || auth.Value != "abc" || auth.Domain != ".deviantart.com" {
		t.Errorf("unexpected auth cookie: %+v", auth)
	}
	if !auth.Secure || !auth.HttpOnly {
		t.Errorf("auth cookie should be Secure+HttpOnly: %+v", auth)
	}
	if auth.Expires.IsZero() || auth.Expires.Unix() != 1893456000 {
		t.Errorf("auth cookie expiry not mapped: %v", auth.Expires)
	}

	if !out[1].Expires.IsZero() {
		t.Errorf("session cookie (Expires<=0) should have zero time, got %v", out[1].Expires)
	}
}
