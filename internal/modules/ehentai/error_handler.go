package ehentai

import (
	"bytes"
	"errors"
	http "github.com/bogdanfinn/fhttp"
	"io"
	"net/url"
	"strings"
)

type IpBanSearchError struct {
}

type IpBanError struct {
}

func (e IpBanSearchError) Error() string {
	return "Content-Length is 0, your IP most likely got banned for searches"
}

func (e IpBanError) Error() string {
	return "Your IP address has been temporarily banned for excessive pageloads"
}

type ErrorHandler struct {
}

func (e ErrorHandler) CheckResponse(response *http.Response) (err error, fatal bool) {
	if response.Header.Get("Content-Length") == "0" {
		return IpBanSearchError{}, true
	}

	content, readErr := io.ReadAll(response.Body)
	if readErr != nil {
		return readErr, true
	}

	if strings.Contains(string(content), "Your IP address has been temporarily banned for excessive pageloads") {
		return IpBanError{}, true
	} else {
		// reset reader for body
		response.Body = io.NopCloser(bytes.NewReader(content))
	}

	return nil, false
}

func (e ErrorHandler) CheckDownloadedFileForErrors(_ int64, _ http.Header) error {
	return nil
}

func (e ErrorHandler) IsFatalError(err error) bool {
	var ue *url.Error
	if errors.As(err, &ue) {
		if ue.Err != nil && ue.Err.Error() == "tls: invalid server key share" {
			return true
		}
	}

	return false
}
