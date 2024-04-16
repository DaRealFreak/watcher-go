package chounyuu

import (
	"github.com/DaRealFreak/watcher-go/internal/http/session"
	"net/http"
)

type DeletedMediaError struct {
	error
}

func (e DeletedMediaError) Error() string {
	return "content got deleted"
}

type errorHandler struct{}

func (e errorHandler) CheckResponse(response *http.Response) (err error, fatal bool) {
	switch {
	case response.StatusCode == 404:
		return DeletedMediaError{}, true
	}

	// fallback to default error handler
	defaultErrorHandler := session.DefaultErrorHandler{}
	return defaultErrorHandler.CheckResponse(response)
}

func (e errorHandler) CheckDownloadedFileForErrors(writtenSize int64, responseHeader http.Header) error {
	// fallback to default error handler
	defaultErrorHandler := session.DefaultErrorHandler{}
	return defaultErrorHandler.CheckDownloadedFileForErrors(writtenSize, responseHeader)
}
