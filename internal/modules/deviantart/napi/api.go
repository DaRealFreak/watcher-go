// Package napi is the implementation of the DeviantArt frontend API
package napi

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	cloudflarebp "github.com/DaRealFreak/cloudflare-bp-go"
	watcherHttp "github.com/DaRealFreak/watcher-go/internal/http"
	"github.com/DaRealFreak/watcher-go/internal/http/session"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"golang.org/x/time/rate"
)

// DeviantartNAPI contains all required items to communicate with the API
type DeviantartNAPI struct {
	UserSession watcherHttp.SessionInterface
	Session     watcherHttp.SessionInterface
	rateLimiter *rate.Limiter
	ctx         context.Context
	account     *models.Account
	moduleKey   string
}

// NewDeviantartNAPI returns the settings of the DeviantArt API
func NewDeviantartNAPI(moduleKey string, account *models.Account) *DeviantartNAPI {
	return &DeviantartNAPI{
		UserSession: session.NewSession(moduleKey),
		Session:     session.NewSession(moduleKey),
		account:     account,
		rateLimiter: rate.NewLimiter(rate.Every(2*time.Second), 1),
		ctx:         context.Background(),
		moduleKey:   moduleKey,
	}
}

// AddRoundTrippers adds the round trippers for CloudFlare, adds a custom user agent
// and implements the implicit OAuth2 authentication and sets the Token round tripper
func (a *DeviantartNAPI) AddRoundTrippers(userAgent string) {
	client := a.UserSession.GetClient()
	// apply CloudFlare bypass
	options := cloudflarebp.GetDefaultOptions()
	if userAgent != "" {
		options.Headers["User-Agent"] = userAgent
	}

	client.Transport = cloudflarebp.AddCloudFlareByPass(client.Transport, options)
	client.Transport = a.setDeviantArtHeaders(client.Transport)
}

// mapAPIResponse maps the API response into the passed APIResponse type
func (a *DeviantartNAPI) mapAPIResponse(res *http.Response, apiRes interface{}) (err error) {
	out, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	content := string(out)

	if res.StatusCode >= 400 {
		var apiErr Error

		if err = json.Unmarshal([]byte(content), &apiErr); err == nil {
			return apiErr
		}

		return fmt.Errorf(`unknown error response: "%s"`, content)
	}

	// unmarshal the request content into the response struct
	if err = json.Unmarshal([]byte(content), &apiRes); err != nil {
		return err
	}

	return nil
}

// applyRateLimit waits until the leaky bucket can pass another request again
func (a *DeviantartNAPI) applyRateLimit() {
	raven.CheckError(a.rateLimiter.Wait(a.ctx))
}
