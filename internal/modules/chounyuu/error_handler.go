package chounyuu

import (
	"github.com/DaRealFreak/watcher-go/internal/http/tls_session"
	http "github.com/bogdanfinn/fhttp"
)

type DeletedMediaError struct {
}

func (e DeletedMediaError) Error() string {
	return "content got deleted"
}

type errorHandler struct{}

func (e errorHandler) CheckResponse(response *http.Response) (err error, fatal bool) {
	switch response.StatusCode {
	case 404:
		return DeletedMediaError{}, true
	}

	// fallback to default error handler
	defaultErrorHandler := tls_session.TlsClientErrorHandler{}
	return defaultErrorHandler.CheckResponse(response)
}

func (e errorHandler) CheckDownloadedFileForErrors(writtenSize int64, responseHeader http.Header) error {
	// fallback to default error handler
	defaultErrorHandler := tls_session.TlsClientErrorHandler{}
	return defaultErrorHandler.CheckDownloadedFileForErrors(writtenSize, responseHeader)
}

func (e errorHandler) IsFatalError(_ error) bool {
	return false
}
