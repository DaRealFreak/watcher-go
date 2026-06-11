package momonga

import (
	http "github.com/bogdanfinn/fhttp"
)

// get issues a GET request with browser-like headers. momon-ga.com rejects requests
// without a recognized browser User-Agent (responds 403), so all page requests have
// to go through here rather than the bare session Get. The image CDN (z*.momon-ga.com)
// does not require this, so the multi-proxy download path is unaffected.
func (m *momonga) get(uri string) (*http.Response, error) {
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, err
	}

	return m.do(req)
}

// do applies the browser-like request headers and dispatches the request on the session
func (m *momonga) do(req *http.Request) (*http.Response, error) {
	m.applyHeaders(req)

	return m.Session.Do(req)
}

// applyHeaders sets the browser-like headers required to avoid momon-ga.com's 403 response
func (m *momonga) applyHeaders(req *http.Request) {
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:135.0) Gecko/20100101 Firefox/135.0")
}
