package napi

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/DaRealFreak/watcher-go/internal/http/session"
)

type DeviantArtErrorHandler struct {
	session.DefaultErrorHandler
	moduleKey string
}

func (e DeviantArtErrorHandler) CheckResponse(response *http.Response) (error error, fatal bool) {
	if response.StatusCode == 403 {
		// 403 is being returned if we're over the rate limit instead of 429
		out, _ := io.ReadAll(response.Body)
		// reset reader for body
		response.Body = io.NopCloser(bytes.NewReader(out))

		// check for cloud front error
		if strings.Contains(string(out), "Generated by cloudfront (CloudFront)") {
			log.WithField("module", e.moduleKey).Warn(
				"ran into 403 error from cloudfront, sleeping 1 minute to recover rate limit",
			)

			time.Sleep(1 * time.Minute)

			return session.StatusError{
				StatusCode: response.StatusCode,
			}, false
		}
	}

	return e.DefaultErrorHandler.CheckResponse(response)
}
