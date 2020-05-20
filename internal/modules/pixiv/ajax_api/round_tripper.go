package ajaxapi

import (
	"fmt"
	browser "github.com/EDDYCJY/fake-useragent"
	"net/http"

	"github.com/DaRealFreak/watcher-go/internal/models"
)

type pixivRoundTripper struct {
	inner         http.RoundTripper
	sessionCookie *models.Cookie
}

// setPixivWebHeaders returns the round tripper for the pixiv web headers
func (a *AjaxAPI) setPixivWebHeaders(inner http.RoundTripper, sessionCookie *models.Cookie) http.RoundTripper {
	return &pixivRoundTripper{
		inner:         inner,
		sessionCookie: sessionCookie,
	}
}

// RoundTrip adds the required request headers to pass CloudFlare checks
func (rt *pixivRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("Accept", "application/json, text/plain, */*")
	r.Header.Set("Accept-Language", "en-US,en;q=0.5")
	r.Header.Set("User-Agent", browser.Firefox())
	r.Header.Set("Referer", "https://www.fanbox.cc")
	r.Header.Set("Origin", "https://www.fanbox.cc")
	r.Header.Set("Cookie", fmt.Sprintf("FANBOXSESSID=%s", rt.sessionCookie.Value))

	if rt.inner == nil {
		return http.DefaultTransport.RoundTrip(r)
	}

	return rt.inner.RoundTrip(r)
}
