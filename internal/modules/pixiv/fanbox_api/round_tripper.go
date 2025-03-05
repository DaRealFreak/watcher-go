package fanboxapi

import (
	"fmt"
	"net/http"

	browser "github.com/EDDYCJY/fake-useragent"

	"github.com/DaRealFreak/watcher-go/internal/models"
)

type pixivRoundTripper struct {
	inner             http.RoundTripper
	sessionCookie     *models.Cookie
	cfClearanceCookie *models.Cookie
	userAgent         string
}

// setPixivWebHeaders returns the round tripper for the pixiv web headers
func (a *FanboxAPI) setPixivWebHeaders(inner http.RoundTripper) http.RoundTripper {
	return &pixivRoundTripper{
		inner:             inner,
		sessionCookie:     a.SessionCookie,
		cfClearanceCookie: a.CfClearanceCookie,
		userAgent:         a.UserAgent,
	}
}

// RoundTrip adds the required request headers to pass CloudFlare checks
func (rt *pixivRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("Accept", "application/json, text/plain, */*")
	r.Header.Set("Accept-Language", "en-US,en;q=0.5")
	r.Header.Set("Referer", "https://www.fanbox.cc")
	r.Header.Set("Origin", "https://www.fanbox.cc")

	// set user agent to the passed user agent (required for most proxies for cf_clearance cookie) or a random one
	// cf_clearance cookie only works in combination with the same IP address and user agent
	if rt.userAgent != "" {
		r.Header.Set("User-Agent", rt.userAgent)
	} else {
		r.Header.Set("User-Agent", browser.Firefox())
	}

	// set the required cookies for the request
	cookieString := fmt.Sprintf("FANBOXSESSID=%s", rt.sessionCookie.Value)
	if rt.cfClearanceCookie != nil {
		cookieString += fmt.Sprintf("; %s=%s", rt.cfClearanceCookie.Name, rt.cfClearanceCookie.Value)
	}
	r.Header.Set("Cookie", cookieString)

	if rt.inner == nil {
		return http.DefaultTransport.RoundTrip(r)
	}

	return rt.inner.RoundTrip(r)
}
