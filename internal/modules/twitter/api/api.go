// Package api is the implementation of the DeviantArt API including the authentication using the Implicit Grant OAuth2
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/dghubble/oauth1"

	watcherHttp "github.com/DaRealFreak/watcher-go/internal/http"
	"github.com/DaRealFreak/watcher-go/internal/http/session"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"golang.org/x/time/rate"
)

// MaxTweetsPerRequest is the maximum amount of tweets you can retrieve from the API in one request
const MaxTweetsPerRequest = 200

// TwitterAPI contains all required items to communicate with the API
type TwitterAPI struct {
	Session     watcherHttp.SessionInterface
	rateLimiter *rate.Limiter
	ctx         context.Context
}

// NewTwitterAPI returns the settings of the Twitter API
func NewTwitterAPI(moduleKey string, oAuth2Client *models.OAuthClient) *TwitterAPI {
	api := &TwitterAPI{
		Session:     session.NewSession(moduleKey),
		rateLimiter: rate.NewLimiter(rate.Every(1*time.Second), 1),
		ctx:         context.Background(),
	}

	config := oauth1.NewConfig(oAuth2Client.ClientID, oAuth2Client.ClientSecret)
	// don't hurt me for abusing refresh token as field for token secret, I don't plan on adding another table just for OAuth1 authentication
	token := oauth1.NewToken(oAuth2Client.AccessToken, oAuth2Client.RefreshToken)

	// add OAuth1 round tripper
	api.Session.SetClient(config.Client(oauth1.NoContext, token))

	return api
}

// mapAPIResponse maps the API response into the passed APIResponse type
func (a *TwitterAPI) mapAPIResponse(res *http.Response, apiRes interface{}) (err error) {
	out, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	content := string(out)

	if res.StatusCode >= 400 {
		var apiErr TwitterError

		if err := json.Unmarshal([]byte(content), &apiErr); err == nil {
			return apiErr
		}

		return fmt.Errorf(`unknown error response: "%s"`, content)
	}

	// unmarshal the request content into the response struct
	if err := json.Unmarshal([]byte(content), &apiRes); err != nil {
		return err
	}

	return nil
}

// applyRateLimit waits until the leaky bucket can pass another request again
func (a *TwitterAPI) applyRateLimit() {
	raven.CheckError(a.rateLimiter.Wait(a.ctx))
}

func (a *TwitterAPI) apiGET(apiRequestURL string, values url.Values) (*http.Response, error) {
	requestURL, _ := url.Parse(apiRequestURL)
	existingValues := requestURL.Query()

	for key, group := range values {
		for _, value := range group {
			existingValues.Add(key, value)
		}
	}

	requestURL.RawQuery = existingValues.Encode()

	return a.Session.Get(requestURL.String())
}
