package api

import (
	http "github.com/bogdanfinn/fhttp"
	"net/url"
	"strings"
)

func (a *SankakuComplexApi) get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.sankaku.api+json;v=2")
	token, err := a.tokenSrc.Token()
	if err == nil {
		req.Header.Set("Authorization", token.TokenType+" "+token.AccessToken)
	}

	return a.Session.Do(req)
}

func (a *SankakuComplexApi) post(url string, data url.Values) (*http.Response, error) {
	formBody := data.Encode()
	req, err := http.NewRequest("POST", url, strings.NewReader(formBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.sankaku.api+json;v=2")
	token, err := a.tokenSrc.Token()
	if err == nil {
		req.Header.Set("Authorization", token.TokenType+" "+token.AccessToken)
	}

	return a.Session.Do(req)
}
