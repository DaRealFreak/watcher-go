// Package pixivapi offers shared functionality for the public API and the mobile API
package pixivapi

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
	"github.com/DaRealFreak/watcher-go/internal/modules/pixiv/pixiv_api/internal"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"golang.org/x/oauth2"
	"golang.org/x/time/rate"
)

// PixivAPI is the struct offering shared functionality for the public API and the mobile API
type PixivAPI struct {
	Session       watcherHttp.SessionInterface
	rateLimiter   *rate.Limiter
	ctx           context.Context
	OAuth2Config  *oauth2.Config
	oAuth2Account *models.OAuthClient
	referer       string
	token         *oauth2.Token
}

// NewPixivAPI returned a pixiv API struct with already configured round trips
func NewPixivAPI(moduleKey string, oAuthClient *models.OAuthClient, referer string) *PixivAPI {
	return &PixivAPI{
		Session: session.NewSession(moduleKey, PixivErrorHandler{}, session.DefaultErrorHandler{}),
		OAuth2Config: &oauth2.Config{
			ClientID:     "MOBrBDS8blbauoSck0ZfDbtuzpyT",
			ClientSecret: "lsACyCD94FhDUtGTXi3QzcFE2uU1hqtDaKeqrdwj",
			Endpoint: oauth2.Endpoint{
				TokenURL: "https://oauth.secure.pixiv.net/auth/token",
			},
		},
		oAuth2Account: oAuthClient,
		referer:       referer,
		rateLimiter:   rate.NewLimiter(rate.Every(2500*time.Millisecond), 1),
		ctx:           context.Background(),
	}
}

// AddRoundTrippers adds the required round trippers for the OAuth2 pixiv APIs
func (a *PixivAPI) AddRoundTrippers() (err error) {
	client := a.Session.GetClient()
	client.Transport = cloudflarebp.AddCloudFlareByPass(client.Transport)
	a.Session.GetClient().Transport = a.setPixivMobileAPIHeaders(client.Transport, a.referer)
	// apply CloudFlare bypass

	if a.token == nil {
		a.token, err = internal.PasswordCredentialsToken(
			a.oAuth2Account, a.OAuth2Config, a.Session.GetClient(),
		)
		if err != nil {
			return err
		}
	}

	httpClientContext := context.WithValue(context.Background(), oauth2.HTTPClient, a.Session.GetClient())

	// set context with own http client for OAuth2 library to use
	// retrieve new client with applied OAuth2 round tripper
	a.Session.SetClient(
		oauth2.NewClient(
			httpClientContext,
			&pixivTokenRefresher{token: a.token, client: a.Session.GetClient(), config: a.OAuth2Config},
		),
	)

	return nil
}

// MapAPIResponse maps the API response into the passed APIResponse type
func (a *PixivAPI) MapAPIResponse(res *http.Response, apiRes interface{}) (err error) {
	out, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	content := string(out)

	if res.StatusCode >= 400 {
		var (
			apiErr         APIError
			apiReqErr      APIRequestError
			mobileAPIError MobileAPIError
		)

		if err = json.Unmarshal([]byte(content), &apiErr); err == nil {
			return apiErr
		}

		if err = json.Unmarshal([]byte(content), &apiReqErr); err == nil {
			return &apiReqErr
		}

		if err = json.Unmarshal([]byte(content), &mobileAPIError); err == nil {
			return &mobileAPIError
		}

		return fmt.Errorf(`unknown error response: "%s"`, content)
	}

	// unmarshal the request content into the response struct
	if err = json.Unmarshal([]byte(content), &apiRes); err != nil {
		return err
	}

	return nil
}

// ApplyRateLimit waits until the leaky bucket can pass another request again
func (a *PixivAPI) ApplyRateLimit() {
	raven.CheckError(a.rateLimiter.Wait(a.ctx))
}
