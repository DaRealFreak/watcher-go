// Package api is the implementation of the DeviantArt API including the authentication using the Implicit Grant OAuth2
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	watcherHttp "github.com/DaRealFreak/watcher-go/pkg/http"
	"github.com/DaRealFreak/watcher-go/pkg/http/session"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
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

	config := &clientcredentials.Config{
		ClientID:     oAuth2Client.ClientID,
		ClientSecret: oAuth2Client.ClientSecret,
		TokenURL:     "https://api.twitter.com/oauth2/token",
	}

	// set context with own http client for OAuth2 library to use
	httpClientContext := context.WithValue(context.Background(), oauth2.HTTPClient, api.Session.GetClient())

	// add OAuth2 round tripper from our client credentials context
	api.Session.SetClient(config.Client(httpClientContext))

	return api
}

// mapAPIResponse maps the API response into the passed APIResponse type
func (a *TwitterAPI) mapAPIResponse(res *http.Response, apiRes interface{}) (err error) {
	out, err := ioutil.ReadAll(res.Body)
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
