package schalenetwork

import (
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/http/tls_session"
	http "github.com/bogdanfinn/fhttp"
)

type schaleErrorHandler struct {
	module *schaleNetwork
}

func (e schaleErrorHandler) isAPIRequest(response *http.Response) bool {
	host := response.Request.URL.Hostname()
	return strings.HasPrefix(host, "api.") || strings.HasPrefix(host, "auth.")
}

func (e schaleErrorHandler) CheckResponse(response *http.Response) (error, bool) {
	if response.StatusCode == 403 && e.isAPIRequest(response) {
		// requests with crt in URL need URL reconstruction after recovery,
		// so return fatal and let inline handling deal with them
		if strings.Contains(response.Request.URL.RawQuery, "crt=") {
			return tls_session.StatusError{StatusCode: 403}, true
		}

		if err := e.module.recoverFrom403(); err != nil {
			return err, true
		}

		// recovery succeeded, return nil so the session retries the request
		return nil, false
	}

	// fallback to default error handler for all other status codes and non-API 403s
	defaultHandler := tls_session.TlsClientErrorHandler{}
	return defaultHandler.CheckResponse(response)
}

func (e schaleErrorHandler) CheckDownloadedFileForErrors(writtenSize int64, responseHeader http.Header) error {
	defaultHandler := tls_session.TlsClientErrorHandler{}
	return defaultHandler.CheckDownloadedFileForErrors(writtenSize, responseHeader)
}

func (e schaleErrorHandler) IsFatalError(_ error) bool {
	return false
}
