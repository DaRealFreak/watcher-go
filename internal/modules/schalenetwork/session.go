package schalenetwork

import (
	http "github.com/bogdanfinn/fhttp"
)

func (m *schaleNetwork) get(uri string) (*http.Response, error) {
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, err
	}

	return m.do(req)
}

func (m *schaleNetwork) post(uri string) (*http.Response, error) {
	req, err := http.NewRequest("POST", uri, nil)
	if err != nil {
		return nil, err
	}

	return m.do(req)
}

func (m *schaleNetwork) do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", m.siteBaseURL()+"/")
	req.Header.Set("Origin", m.siteBaseURL())
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", m.secFetchSite())

	if m.settings.Cloudflare.UserAgent != "" {
		req.Header.Set("User-Agent", m.settings.Cloudflare.UserAgent)
	}

	return m.Session.Do(req)
}
