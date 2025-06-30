package graphql_api

import (
	"github.com/DaRealFreak/watcher-go/internal/modules/twitter/graphql_api/xpff"
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

	req.Header.Set("content-type", "application/x-www-form-urlencoded")

	return a.apiDo(req)
}

func (a *TwitterGraphQlAPI) apiDo(req *http.Request) (*http.Response, error) {
	// set static headers (User-Agent, Referer, etc.)
	req.Header.Set("User-Agent", a.settings.UserAgent)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("content-type", "application/json")
	req.Header.Set("Referer", "https://x.com/")

	req.Header.Set("authorization", "Bearer AAAAAAAAAAAAAAAAAAAAANRILgAAAAAAnNwIzUejRCOuH5E6I8xnZz4puTs=1Zv7ttfk8LF81IUq16cHjhLTvJu4FA33AGWWjCpTnA")
	req.Header.Set("x-twitter-auth-type", "OAuth2Session")
	req.Header.Set("x-twitter-client-language", "en")
	req.Header.Set("x-twitter-active-user", "yes")

	// 2) Extract cookies, build the Cookie header and x-csrf-token
	var csrfToken string
	for _, c := range a.Session.GetClient().GetCookies(req.URL) {
		switch c.Name {
		case "ct0":
			csrfToken = c.Value
		case "guest_id":
			if a.xpffHandler == nil {
				a.xpffHandler = xpff.NewHandler(c.Value, a.settings.UserAgent)
			}
		}
	}

	// set the csrf token header if available
	if csrfToken != "" {
		req.Header.Set("x-csrf-token", csrfToken)
	}

	// generate and set x-client-transaction-id header
	txID, err := a.xTransactionIdHandler.GenerateTransactionId(
		req.Method, req.URL.Path, nil, "", "", 0, 0,
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-client-transaction-id", txID)

	// add XPFF header if available
	if a.xpffHandler != nil {
		xpffHdr, err := a.xpffHandler.GetXPFFHeader()
		if err != nil {
			return nil, err
		}
		req.Header.Set("x-xp-forwarded-for", xpffHdr)
	}

	return a.Session.Do(req)
}
