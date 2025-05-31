package napi

import (
	http2 "github.com/DaRealFreak/watcher-go/internal/http"
	http "github.com/bogdanfinn/fhttp"
	"net/url"
	"strings"
)

func (a *DeviantartNAPI) get(url string, session ...http2.SessionInterface) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	return a.do(req, session...)
}

func (a *DeviantartNAPI) post(url string, data url.Values, session ...http2.SessionInterface) (*http.Response, error) {
	formBody := data.Encode()
	req, err := http.NewRequest("POST", url, strings.NewReader(formBody))
	if err != nil {
		return nil, err
	}

	return a.do(req, session...)
}

func (a *DeviantartNAPI) do(req *http.Request, session ...http2.SessionInterface) (*http.Response, error) {
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("User-Agent", a.UserAgent)

	if req.URL.Host == "www.deviantart.com" {
		req.Header.Set("Host", "https://www.deviantart.com")
		req.Header.Set("Referer", "https://www.deviantart.com")
		req.Header.Set("Origin", "https://www.deviantart.com")
	}

	usedSession := a.UserSession
	if len(session) > 0 {
		usedSession = session[0]
	}

	return usedSession.Do(req)
}
