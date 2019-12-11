package ajaxapi

import (
	"fmt"
	"net/http"

	browser "github.com/EDDYCJY/fake-useragent"
)

type pixivRoundTripper struct {
	inner     http.RoundTripper
	loginData LoginData
}

// LoginData contains the required cookie data which is additionally added in the header
type LoginData struct {
	SessionID   string
	DeviceToken string
}

// SetPixivWebHeaders returns the round tripper for the pixiv web headers
func SetPixivWebHeaders(inner http.RoundTripper, loginData LoginData) http.RoundTripper {
	return &pixivRoundTripper{
		inner:     inner,
		loginData: loginData,
	}
}

// RoundTrip adds the required request headers to pass CloudFlare checks
func (rt *pixivRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	r.Header.Set("Accept-Encoding", "gzip, deflate, br")
	r.Header.Set("Accept-Language", "en-US,en;q=0.5")
	r.Header.Set("User-Agent", browser.Firefox())
	r.Header.Set("Referer", "https://www.pixiv.net/")
	r.Header.Set("Origin", "https://www.pixiv.net")
	r.Header.Set(
		"Cookie",
		fmt.Sprintf("PHPSESSID=%s; device_token=%s", rt.loginData.SessionID, rt.loginData.DeviceToken),
	)

	if rt.inner == nil {
		return http.DefaultTransport.RoundTrip(r)
	}

	return rt.inner.RoundTrip(r)
}
