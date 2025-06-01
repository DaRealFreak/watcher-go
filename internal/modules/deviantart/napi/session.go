package napi

import (
	http2 "github.com/DaRealFreak/watcher-go/internal/http"
	http "github.com/bogdanfinn/fhttp"
)

func (a *DeviantartNAPI) get(url string, session ...http2.SessionInterface) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	return a.do(req, session...)
}

func (a *DeviantartNAPI) do(req *http.Request, session ...http2.SessionInterface) (*http.Response, error) {
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("User-Agent", a.UserAgent)

	if req.URL.Host == "www.deviantart.com" {
		if req.Header.Get("Host") == "" {
			req.Header.Set("Host", "www.deviantart.com")
		}

		if req.Header.Get("Referer") == "" {
			req.Header.Set("Referer", "https://www.deviantart.com")
		}

		if req.Header.Get("Origin") == "" {
			req.Header.Set("Origin", "https://www.deviantart.com")
		}
	}

	usedSession := a.UserSession
	if len(session) > 0 {
		usedSession = session[0]
	}

	return usedSession.Do(req)
}
