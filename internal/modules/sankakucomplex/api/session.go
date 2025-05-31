package api

import (
	http "github.com/bogdanfinn/fhttp"
)

func (a *SankakuComplexApi) get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	return a.do(req)
}

func (a *SankakuComplexApi) do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Accept", "application/vnd.sankaku.api+json;v=2")
	token, err := a.tokenSrc.Token()
	if err == nil {
		req.Header.Set("Authorization", token.TokenType+" "+token.AccessToken)
	}

	return a.Session.Do(req)
}
