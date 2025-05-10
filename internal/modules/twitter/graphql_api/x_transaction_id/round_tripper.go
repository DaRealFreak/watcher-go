package x_transaction_id

import (
	"net/http"
)

type twitterRoundTripper struct {
	inner  http.RoundTripper
	client *http.Client
}

func (h *XTransactionIdHandler) setTwitterAPIHeaders(client *http.Client) http.RoundTripper {
	rt := &twitterRoundTripper{
		inner:  client.Transport,
		client: client,
	}

	client.Transport = rt

	return rt
}

// RoundTrip adds the required request headers to pass server side checks of twitter
func (rt *twitterRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("Accept-Language", "en-US,en;q=0.5")
	r.Header.Set("Cache-Control", "no-cache")
	r.Header.Set("Refer", "https://x.com/")
	r.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:138.0) Gecko/20100101 Firefox/138.0")

	if rt.inner == nil {
		return http.DefaultTransport.RoundTrip(r)
	}

	return rt.inner.RoundTrip(r)
}
