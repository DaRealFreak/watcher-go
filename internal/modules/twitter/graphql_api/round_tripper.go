package graphql_api

import (
	"net/http"
	"net/url"
)

type twitterGraphQLApiRoundTripper struct {
	inner  http.RoundTripper
	client *http.Client
}

func (a *TwitterGraphQlAPI) setTwitterAPIHeaders(client *http.Client) http.RoundTripper {
	rt := &twitterGraphQLApiRoundTripper{
		inner:  client.Transport,
		client: client,
	}

	client.Transport = rt

	return rt
}

// RoundTrip adds the required request headers to pass server side checks of pixiv
func (rt *twitterGraphQLApiRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("authorization", "Bearer AAAAAAAAAAAAAAAAAAAAANRILgAAAAAAnNwIzUejRCOuH5E6I8xnZz4puTs%3D1Zv7ttfk8LF81IUq16cHjhLTvJu4FA33AGWWjCpTnA")
	r.Header.Set("x-twitter-active-user", "yes")
	r.Header.Set("x-twitter-auth-type", "OAuth2Session")
	r.Header.Set("x-csrf-token", "")
	r.Header.Set("x-twitter-client-language", "en")
	r.Header.Set("Referer", "https://x.com/")

	requestUrl, _ := url.Parse(r.URL.String())
	cookies := rt.client.Jar.Cookies(requestUrl)
	for _, cookie := range cookies {
		if cookie.Name == "ct0" {
			r.Header.Set("x-csrf-token", cookie.Value)
		}
	}

	if rt.inner == nil {
		return http.DefaultTransport.RoundTrip(r)
	}

	return rt.inner.RoundTrip(r)
}
