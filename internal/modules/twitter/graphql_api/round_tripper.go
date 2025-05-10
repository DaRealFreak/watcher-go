package graphql_api

import (
	"net/http"
	"net/url"
)

type twitterGraphQLApiRoundTripper struct {
	inner             http.RoundTripper
	client            *http.Client
	nextTransactionId string
}

func (a *TwitterGraphQlAPI) setTwitterAPIHeaders(client *http.Client) *twitterGraphQLApiRoundTripper {
	rt := &twitterGraphQLApiRoundTripper{
		inner:  client.Transport,
		client: client,
	}

	client.Transport = rt

	return rt
}

// RoundTrip adds the required request headers to pass server side checks of pixiv
func (rt *twitterGraphQLApiRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:138.0) Gecko/20100101 Firefox/138.0")
	r.Header.Set("Accept-Language", "en-US,en;q=0.5")
	r.Header.Set("Referer", "https://x.com/")
	r.Header.Set("content-type", "application/json")
	r.Header.Set("x-twitter-auth-type", "OAuth2Session")
	r.Header.Set("x-twitter-active-user", "yes")
	r.Header.Set("authorization", "Bearer AAAAAAAAAAAAAAAAAAAAANRILgAAAAAAnNwIzUejRCOuH5E6I8xnZz4puTs%3D1Zv7ttfk8LF81IUq16cHjhLTvJu4FA33AGWWjCpTnA")
	r.Header.Set("x-twitter-client-language", "en")

	requestUrl, _ := url.Parse(r.URL.String())
	cookies := rt.client.Jar.Cookies(requestUrl)
	for _, cookie := range cookies {
		if cookie.Name == "ct0" {
			r.Header.Set("x-csrf-token", cookie.Value)
		}
	}

	if rt.nextTransactionId != "" {
		r.Header.Set("x-client-transaction-id", rt.nextTransactionId)
		rt.nextTransactionId = ""
	}

	//println("current request")
	//println("URL: ", r.URL.String())
	//println("X-CSRF-TOKEN: ", r.Header.Get("x-csrf-token"))
	//println("X-CLIENT-TRANSACTION-ID: ", r.Header.Get("x-client-transaction-id"))

	if rt.inner == nil {
		return http.DefaultTransport.RoundTrip(r)
	}

	return rt.inner.RoundTrip(r)
}
