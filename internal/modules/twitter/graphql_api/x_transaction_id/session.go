package x_transaction_id

import (
	http "github.com/bogdanfinn/fhttp"
	"net/url"
	"strings"
)

func (h *XTransactionIdHandler) get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Refer", "https://x.com/")
	req.Header.Set("X-Twitter-Active-User", "yes")
	req.Header.Set("X-Twitter-Client-Language", "en")
	req.Header.Set("User-Agent", h.settings.UserAgent)

	return h.transactionSession.GetClient().Do(req)
}

func (h *XTransactionIdHandler) post(url string, data url.Values) (*http.Response, error) {
	formBody := data.Encode()
	req, err := http.NewRequest("POST", url, strings.NewReader(formBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Refer", "https://x.com/")
	req.Header.Set("X-Twitter-Active-User", "yes")
	req.Header.Set("X-Twitter-Client-Language", "en")
	req.Header.Set("User-Agent", h.settings.UserAgent)
	req.Header.Set("Content-Type", "application/json")

	return h.transactionSession.GetClient().Do(req)
}
