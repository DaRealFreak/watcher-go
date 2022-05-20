// Package graphql_api is the implementation of the public API (be aware it's against Twitters ToS to use it)
package graphql_api

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	watcherHttp "github.com/DaRealFreak/watcher-go/internal/http"
	"github.com/DaRealFreak/watcher-go/internal/http/session"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"golang.org/x/time/rate"
)

// TwitterGraphQlAPI contains all required items to communicate with the GraphQL API
type TwitterGraphQlAPI struct {
	Session     watcherHttp.SessionInterface
	rateLimiter *rate.Limiter
	ctx         context.Context
}

// NewTwitterAPI returns the settings of the Twitter API
func NewTwitterAPI(moduleKey string) *TwitterGraphQlAPI {
	return &TwitterGraphQlAPI{
		Session:     session.NewSession(moduleKey),
		rateLimiter: rate.NewLimiter(rate.Every(1*time.Second), 1),
		ctx:         context.Background(),
	}
}

// mapAPIResponse maps the API response into the passed APIResponse type
func (a *TwitterGraphQlAPI) mapAPIResponse(res *http.Response, apiRes interface{}) (err error) {
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
func (a *TwitterGraphQlAPI) applyRateLimit() {
	raven.CheckError(a.rateLimiter.Wait(a.ctx))
}

func (a *TwitterGraphQlAPI) apiGET(apiRequestURL string, values url.Values) (*http.Response, error) {
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
