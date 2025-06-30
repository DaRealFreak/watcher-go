// Package graphql_api is the implementation of the public API (be aware it's against Twitters ToS to use it)
package graphql_api

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/DaRealFreak/watcher-go/internal/http/session"
	"github.com/DaRealFreak/watcher-go/internal/modules/twitter/graphql_api/x_transaction_id"
	"github.com/DaRealFreak/watcher-go/internal/modules/twitter/graphql_api/xpff"
	"github.com/DaRealFreak/watcher-go/internal/modules/twitter/twitter_settings"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	http "github.com/bogdanfinn/fhttp"
	"io"
	"net/url"
	"time"

	watcherHttp "github.com/DaRealFreak/watcher-go/internal/http"
	log "github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

const CookieAuth = "auth_token"

// TwitterGraphQlAPI contains all required items to communicate with the GraphQL API
type TwitterGraphQlAPI struct {
	settings               twitter_settings.TwitterSettings
	xTransactionIdHandler  *x_transaction_id.XTransactionIdHandler
	xpffHandler            *xpff.Handler
	authTokenFallbackIndex int
	moduleKey              string
	Session                watcherHttp.SessionInterface
	rateLimiter            *rate.Limiter
	ctx                    context.Context
}

// NewTwitterAPI returns the settings of the Twitter API
func NewTwitterAPI(moduleKey string, settings twitter_settings.TwitterSettings, proxySettings *watcherHttp.ProxySettings) *TwitterGraphQlAPI {
	graphQLSession := session.NewSession(moduleKey, TwitterErrorHandler{})
	raven.CheckError(graphQLSession.SetProxy(proxySettings))

	return &TwitterGraphQlAPI{
		settings:               settings,
		xTransactionIdHandler:  x_transaction_id.NewXTransactionIdHandler(graphQLSession, settings),
		authTokenFallbackIndex: 0,
		moduleKey:              moduleKey,
		Session:                graphQLSession,
		rateLimiter:            rate.NewLimiter(rate.Every(5000*time.Millisecond), 1),
		ctx:                    context.Background(),
	}
}

// SetCookies adds the passed cookies to both sessions, the graphql and the x-transaction id session
func (a *TwitterGraphQlAPI) SetCookies(cookies []*http.Cookie) {
	requestUrl, _ := url.Parse("https://x.com/")
	a.Session.GetClient().SetCookies(
		requestUrl,
		cookies,
	)
}

// mapAPIResponse maps the API response into the passed APIResponse type
func (a *TwitterGraphQlAPI) mapAPIResponse(res *http.Response, apiRes interface{}) (err error) {
	out, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	content := string(out)

	if res.StatusCode >= 400 {
		var apiErr TwitterError

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
func (a *TwitterGraphQlAPI) applyRateLimit() {
	raven.CheckError(a.rateLimiter.Wait(a.ctx))
}

func (a *TwitterGraphQlAPI) handleGetRequest(apiRequestURL string, values url.Values) (*http.Response, error) {
	requestURL, _ := url.Parse(apiRequestURL)
	existingValues := requestURL.Query()

	for key, group := range values {
		for _, value := range group {
			existingValues.Add(key, value)
		}
	}

	requestURL.RawQuery = existingValues.Encode()
	res, err := a.apiGet(requestURL.String())
	if err != nil {
		switch err.(type) {
		case SessionTerminatedError:
			// try to use a fallback auth token if available
			if a.authTokenFallbackIndex < len(a.settings.FallbackAuthTokens) {
				// inform user about the session termination
				log.WithField("module", a.moduleKey).Warnf(
					"received 401 status code for URI \"%s\", session got probably terminated",
					requestURL.String(),
				)

				twitterURL, _ := url.Parse("https://x.com")
				currentCookies := a.Session.GetCookies(twitterURL)
				for _, cookie := range currentCookies {
					if cookie.Name == "auth_token" {
						cookie.Value = a.settings.FallbackAuthTokens[a.authTokenFallbackIndex]
						a.authTokenFallbackIndex++

						// update cookie with new value for the session
						a.Session.SetCookies(twitterURL, []*http.Cookie{cookie})
						break
					}
				}

				return a.handleGetRequest(apiRequestURL, values)
			}
		case SessionRefreshError:
			log.WithField("module", a.moduleKey).Warnf(
				"received 404 status code for URI \"%s\", session probably needs a refresh after 429 status code",
				requestURL.String(),
			)
			if err = a.InitializeSession(); err != nil {
				return nil, err
			}

			return a.handleGetRequest(apiRequestURL, values)
		}
	}

	return res, err
}

func (a *TwitterGraphQlAPI) handlePostRequest(apiRequestURL string, values url.Values) (*http.Response, error) {
	requestURL, _ := url.Parse(apiRequestURL)
	res, err := a.apiPost(requestURL.String(), values)
	if err != nil {
		switch err.(type) {
		case SessionTerminatedError:
			// try to use a fallback auth token if available
			if a.authTokenFallbackIndex < len(a.settings.FallbackAuthTokens) {
				// inform user about the session termination
				log.WithField("module", a.moduleKey).Warnf(
					"received 401 status code for URI \"%s\", session got probably terminated",
					requestURL.String(),
				)

				twitterURL, _ := url.Parse("https://x.com")
				currentCookies := a.Session.GetCookies(twitterURL)
				for _, cookie := range currentCookies {
					if cookie.Name == "auth_token" {
						cookie.Value = a.settings.FallbackAuthTokens[a.authTokenFallbackIndex]
						a.authTokenFallbackIndex++

						// update cookie with new value for the session
						a.SetCookies([]*http.Cookie{cookie})
						break
					}
				}

				return a.handlePostRequest(apiRequestURL, values)
			}
		case SessionRefreshError:
			log.WithField("module", a.moduleKey).Warnf(
				"received 404 status code for URI \"%s\", session probably needs a refresh after 429 status code",
				requestURL.String(),
			)
			if err = a.InitializeSession(); err != nil {
				return nil, err
			}

			return a.handlePostRequest(apiRequestURL, values)
		}
	}

	return res, err
}

func (a *TwitterGraphQlAPI) InitializeSession() error {
	err := a.xTransactionIdHandler.ExtractAnimationKey()
	if err != nil {
		return err
	}

	return nil
}
