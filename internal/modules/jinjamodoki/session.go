package jinjamodoki

import (
	http "github.com/bogdanfinn/fhttp"
	"net/url"
	"strings"
)

func (m *jinjaModoki) get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	return m.do(req)
}

func (m *jinjaModoki) post(url string, data url.Values) (*http.Response, error) {
	formBody := data.Encode()
	req, err := http.NewRequest("POST", url, strings.NewReader(formBody))
	if err != nil {
		return nil, err
	}

	return m.do(req)
}

func (m *jinjaModoki) do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Origin", "https://gs-uploader.jinja-modoki.com")
	req.Header.Set("Referer", "https://gs-uploader.jinja-modoki.com/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:138.0) Gecko/20100101 Firefox/138.0")

	return m.Session.Do(req)
}
