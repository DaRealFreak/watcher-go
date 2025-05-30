package api

import (
	"fmt"
	browser "github.com/EDDYCJY/fake-useragent"
	http "github.com/bogdanfinn/fhttp"
	"net/url"
	"strings"
)

func (a *ChounyuuAPI) Get(requestUrl string) (*http.Response, error) {
	req, err := http.NewRequest("GET", requestUrl, nil)
	if err != nil {
		return nil, err
	}

	domain := ChounyuuDomain
	parsedUrl, _ := url.Parse(requestUrl)
	if strings.Contains(parsedUrl.Host, SuperFutaDomain) {
		domain = SuperFutaDomain
	}

	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Origin", fmt.Sprintf("https://g.%s", domain))
	req.Header.Set("Referer", fmt.Sprintf("https://g.%s", domain))
	req.Header.Set("User-Agent", browser.Firefox())

	return a.Session.Do(req)
}

func (a *ChounyuuAPI) Do(req *http.Request) (*http.Response, error) {
	domain := ChounyuuDomain
	if strings.Contains(req.URL.Host, SuperFutaDomain) {
		domain = SuperFutaDomain
	}

	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Origin", fmt.Sprintf("https://g.%s", domain))
	req.Header.Set("Referer", fmt.Sprintf("https://g.%s", domain))
	req.Header.Set("User-Agent", browser.Firefox())

	return a.Session.Do(req)
}
