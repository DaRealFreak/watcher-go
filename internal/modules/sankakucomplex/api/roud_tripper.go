package api

import (
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
)

type sankakuComplexAuthorizationRoundTripper struct {
	inner    http.RoundTripper
	tokenSrc oauth2.TokenSource
}

// setDeviantArtHeaders returns the round tripper for the DeviantArt console API headers
func (a *SankakuComplexApi) addRoundTripper(inner http.RoundTripper) http.RoundTripper {
	return &sankakuComplexAuthorizationRoundTripper{
		inner:    inner,
		tokenSrc: a.tokenSrc,
	}
}

// RoundTrip adds the required request headers to pass server side checks of DeviantArt
func (rt *sankakuComplexAuthorizationRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	token, err := rt.tokenSrc.Token()
	if err == nil {
		r.Header.Set("Authorization", fmt.Sprintf("%s %s", token.TokenType, token.AccessToken))
	}

	if r.Header.Get("Accept") == "" {
		r.Header.Set("Accept", "application/vnd.sankaku.api+json;v=2")
	}

	if rt.inner == nil {
		return http.DefaultTransport.RoundTrip(r)
	}

	return rt.inner.RoundTrip(r)
}
