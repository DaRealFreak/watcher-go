package patreon

import (
	http "github.com/bogdanfinn/fhttp"
)

func (m *patreon) get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	return m.do(req)
}

func (m *patreon) do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("User-Agent", "Patreon/7.6.28 (Android; Android 11; Scale/2.10)")

	return m.Session.Do(req)
}
