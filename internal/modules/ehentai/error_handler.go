package ehentai

import (
	"net/http"
)

type IpBanSearchError struct {
	error
}

func (e IpBanSearchError) Error() string {
	return "Content-Length is 0, your IP most likely got banned for searches"
}

type ErrorHandler struct {
}

func (e ErrorHandler) CheckResponse(response *http.Response) (err error, fatal bool) {
	if response.Header.Get("Content-Length") == "0" {
		return IpBanSearchError{}, true
	}

	return nil, false
}

func (e ErrorHandler) CheckDownloadedFileForErrors(writtenSize int64, responseHeader http.Header) error {
	return nil
}
