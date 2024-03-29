package ehentai

import (
	"bytes"
	"io"
	"net/http"
	"strings"
)

type IpBanSearchError struct {
	error
}

type IpBanError struct {
	error
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
