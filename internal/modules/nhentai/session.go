package nhentai

import (
	http "github.com/bogdanfinn/fhttp"
	"net/url"
	"strings"
)

func (m *nhentai) get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Origin", url)
	req.Header.Set("Referer", url)
	req.Header.Set("User-Agent", m.settings.Cloudflare.UserAgent)

	return m.Session.Do(req)
}

func (m *nhentai) post(url string, data url.Values) (*http.Response, error) {
	formBody := data.Encode()
	req, err := http.NewRequest("POST", url, strings.NewReader(formBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Origin", url)
	req.Header.Set("Referer", url)
	req.Header.Set("User-Agent", m.settings.Cloudflare.UserAgent)

	return m.Session.Do(req)
}
