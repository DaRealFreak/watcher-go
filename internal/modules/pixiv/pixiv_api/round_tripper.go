package pixivapi

import (
	// nolint: gosec
	"crypto/md5"
	"fmt"
	"net/http"
	"time"

	browser "github.com/EDDYCJY/fake-useragent"
)

type pixivPublicAPIRoundTripper struct {
	inner   http.RoundTripper
	referer string
}

// SetPixivMobileAPIHeaders returns the round tripper for the pixiv mobile API headers
func (a *PixivAPI) setPixivMobileAPIHeaders(inner http.RoundTripper, referer string) http.RoundTripper {
	return &pixivPublicAPIRoundTripper{
		inner:   inner,
		referer: referer,
	}
}

// RoundTrip adds the required request headers to pass server side checks of pixiv
func (rt *pixivPublicAPIRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("Accept-Language", "en_US")
	r.Header.Set("App-OS", "android")
	r.Header.Set("App-OS-Version", "9")
	r.Header.Set("App-Version", "5.0.156")
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Referer", rt.referer)
	r.Header.Set("User-Agent", browser.Firefox())

	// add X-Client-Time and X-Client-Hash which are now getting validated server side
	localTime := time.Now()
	r.Header.Add("X-Client-Time", localTime.Format(time.RFC3339))
	r.Header.Add("X-Client-Hash", fmt.Sprintf(
		// nolint: gosec
		"%x", md5.Sum(
			[]byte(localTime.Format(time.RFC3339)+"28c1fdd170a5204386cb1313c7077b34f83e4aaf4aa829ce78c231e05b0bae2c"),
		),
	))

	if rt.inner == nil {
		return http.DefaultTransport.RoundTrip(r)
	}

	return rt.inner.RoundTrip(r)
}
