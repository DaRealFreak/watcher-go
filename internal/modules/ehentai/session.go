package ehentai

import (
	http2 "github.com/DaRealFreak/watcher-go/internal/http"
	browser "github.com/EDDYCJY/fake-useragent"
	http "github.com/bogdanfinn/fhttp"
	"net/url"
	"strings"
)

func (m *ehentai) get(requestUrl string, session ...http2.SessionInterface) (*http.Response, error) {
	req, err := http.NewRequest("GET", requestUrl, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Origin", requestUrl)
	req.Header.Set("Referer", requestUrl)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:131.0) Gecko/20100101 Firefox/131.0")

	usedSession := m.Session
	if len(session) > 0 {
		usedSession = session[0]
	}

	return usedSession.Do(req)
}

func (m *ehentai) post(requestUrl string, data url.Values, session ...http2.SessionInterface) (*http.Response, error) {
	formBody := data.Encode()
	req, err := http.NewRequest("POST", requestUrl, strings.NewReader(formBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Origin", requestUrl)
	req.Header.Set("Referer", requestUrl)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:131.0) Gecko/20100101 Firefox/131.0")

	usedSession := m.Session
	if len(session) > 0 {
		usedSession = session[0]
	}

	return usedSession.GetClient().Do(req)
}

func (m *ehentai) do(req *http.Request, session ...http2.SessionInterface) (*http.Response, error) {
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
