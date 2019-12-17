// Package pixivapi offers shared functionality for the public API and the mobile API
package pixivapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	watcherHttp "github.com/DaRealFreak/watcher-go/pkg/http"
	"github.com/DaRealFreak/watcher-go/pkg/http/session"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	"golang.org/x/oauth2"
)

// PixivAPI is the struct offering shared functionality for the public API and the mobile API
type PixivAPI struct {
	Session      watcherHttp.SessionInterface
	OAuth2Config *oauth2.Config
}

// NewPixivAPI returned a pixiv API struct with already configured round trips
func NewPixivAPI(moduleKey string, account *models.Account, referer string) (PixivAPI, error) {
	pixivAPI := PixivAPI{
		Session: session.NewSession(moduleKey),
		OAuth2Config: &oauth2.Config{
			ClientID:     "MOBrBDS8blbauoSck0ZfDbtuzpyT",
			ClientSecret: "lsACyCD94FhDUtGTXi3QzcFE2uU1hqtDaKeqrdwj",
			Endpoint: oauth2.Endpoint{
				TokenURL: "https://oauth.secure.pixiv.net/auth/token",
			},
		},
	}

	client := pixivAPI.Session.GetClient()
	client.Transport = pixivAPI.setPixivMobileAPIHeaders(client.Transport, referer)
	httpClientContext := context.WithValue(context.Background(), oauth2.HTTPClient, client)

	token, err := pixivAPI.passwordCredentialsToken(account.Username, account.Password)
	if err != nil {
		return pixivAPI, err
	}

	// set context with own http client for OAuth2 library to use
	// retrieve new client with applied OAuth2 round tripper
	client = pixivAPI.OAuth2Config.Client(httpClientContext, token)

	pixivAPI.Session.SetClient(client)

	return pixivAPI, nil
}

// MapAPIResponse maps the API response into the passed APIResponse type
func (a *PixivAPI) MapAPIResponse(res *http.Response, apiRes interface{}) (err error) {
	content := a.Session.GetDocument(res).Text()

	if res.StatusCode >= 400 {
		var (
			apiErr    APIError
			apiReqErr APIRequestError
		)

		if err := json.Unmarshal([]byte(content), &apiErr); err == nil {
			return apiErr
		}

		if err := json.Unmarshal([]byte(content), &apiReqErr); err == nil {
			return &apiReqErr
		}

		return fmt.Errorf(`unknown error response: "%s"`, content)
	}

	// unmarshal the request content into the response struct
	if err := json.Unmarshal([]byte(content), &apiRes); err != nil {
		return err
	}

	return nil
}
