package graphql_api

import (
	http "github.com/bogdanfinn/fhttp"
	"net/url"
	"strings"
)

func (a *TwitterGraphQlAPI) apiGet(requestUrl string) (*http.Response, error) {
	req, err := http.NewRequest("GET", requestUrl, nil)
	if err != nil {
		return nil, err
	}

	return a.apiDo(req)
}

func (a *TwitterGraphQlAPI) apiPost(requestUrl string, values url.Values) (*http.Response, error) {
	req, err := http.NewRequest("POST", requestUrl, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}

	return a.apiDo(req)
}

func (a *TwitterGraphQlAPI) apiDo(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:138.0) Gecko/20100101 Firefox/138.0")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Referer", "https://x.com/")
	req.Header.Set("content-type", "application/x-www-form-urlencoded")
	req.Header.Set("x-twitter-auth-type", "OAuth2Session")
	req.Header.Set("x-twitter-active-user", "yes")
	req.Header.Set("authorization", "Bearer AAAAAAAAAAAAAAAAAAAAANRILgAAAAAAnNwIzUejRCOuH5E6I8xnZz4puTs%3D1Zv7ttfk8LF81IUq16cHjhLTvJu4FA33AGWWjCpTnA")
	req.Header.Set("x-twitter-client-language", "en")

	cookies := a.Session.GetClient().GetCookies(req.URL)
	for _, cookie := range cookies {
		if cookie.Name == "ct0" {
			req.Header.Set("x-csrf-token", cookie.Value)
			break
		}
	}

	xTransactionId, err := a.xTransactionIdHandler.GenerateTransactionId(
		"POST",
		strings.TrimPrefix(req.URL.String(), "https://x.com"),
		nil, "", "", 0, 0,
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("x-twitter-client-transaction-id", xTransactionId)

	return a.Session.Do(req)
}
