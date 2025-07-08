package ehentai

import (
	http2 "github.com/DaRealFreak/watcher-go/internal/http"
	browser "github.com/EDDYCJY/fake-useragent"
	"net/http"
	"net/url"
	"strings"
)

func (m *ehentai) get(requestUrl string, session ...http2.StdClientSessionInterface) (*http.Response, error) {
	req, err := http.NewRequest("GET", requestUrl, nil)
	if err != nil {
		return nil, err
	}

	return m.do(req, session...)
}

func (m *ehentai) post(requestUrl string, data url.Values, session ...http2.StdClientSessionInterface) (*http.Response, error) {
	formBody := data.Encode()
	req, err := http.NewRequest("POST", requestUrl, strings.NewReader(formBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return m.do(req, session...)
}

func (m *ehentai) do(req *http.Request, session ...http2.StdClientSessionInterface) (*http.Response, error) {
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Origin", req.URL.String())
	req.Header.Set("Referer", req.URL.String())
	req.Header.Set("User-Agent", browser.Firefox())

	usedSession := m.Session
	if len(session) > 0 {
		usedSession = session[0]
	}

	return usedSession.Do(req)
}
