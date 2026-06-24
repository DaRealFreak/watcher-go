package api

import (
	"fmt"

	browser "github.com/EDDYCJY/fake-useragent"
	http "github.com/bogdanfinn/fhttp"
)

func (api *Client) Get(requestUrl string) (*http.Response, error) {
	req, err := http.NewRequest("GET", requestUrl, nil)
	if err != nil {
		return nil, err
	}

	// kept consistent with the kemono module; harmless on pawchive's Cloudflare
	// front (plain JSON also works) but keeps both modules behaving identically.
	req.Header.Set("Accept", "text/css")

	return api.Do(req)
}

func (api *Client) Do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")
	req.Header.Set("Origin", fmt.Sprintf("https://%s", req.URL.Host))
	req.Header.Set("Referer", fmt.Sprintf("https://%s", req.URL.Host))
	req.Header.Set("User-Agent", browser.Firefox())

	return api.Client.Do(req)
}
