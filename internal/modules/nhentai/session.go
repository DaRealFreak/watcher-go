package nhentai

import (
	http "github.com/bogdanfinn/fhttp"
)

func (m *nhentai) get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	return m.do(req)
}

func (m *nhentai) do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Origin", req.URL.String())
	req.Header.Set("Referer", req.URL.String())
	req.Header.Set("User-Agent", m.settings.Cloudflare.UserAgent)

	return m.Session.Do(req)
}
