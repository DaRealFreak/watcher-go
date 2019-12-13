// Package mobileapi handles the default API functionality reverse engineered from the mobile application
// since the API is not documented or intended to be used outside of the mobile application
package mobileapi

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

// MobileAPI is the implementation of the API used in the mobile applications
type MobileAPI struct {
	Session      watcherHttp.SessionInterface
	OAuth2Config *oauth2.Config
}

// NewMobileAPI initializes the mobile API and handles the whole OAuth2 and round tripper procedures
func NewMobileAPI(moduleKey string, account models.Account) (*MobileAPI, error) {
	mobileAPI := &MobileAPI{
		Session: session.NewSession(moduleKey),
		OAuth2Config: &oauth2.Config{
			ClientID:     "MOBrBDS8blbauoSck0ZfDbtuzpyT",
			ClientSecret: "lsACyCD94FhDUtGTXi3QzcFE2uU1hqtDaKeqrdwj",
			Endpoint: oauth2.Endpoint{
				TokenURL: "https://oauth.secure.pixiv.net/auth/token",
			},
		},
	}

	client := mobileAPI.Session.GetClient()
	client.Transport = mobileAPI.setPixivMobileAPIHeaders(client.Transport)
	httpClientContext := context.WithValue(context.Background(), oauth2.HTTPClient, client)

	token, err := mobileAPI.passwordCredentialsToken(account.Username, account.Password)
	if err != nil {
		return nil, err
	}

	// set context with own http client for OAuth2 library to use
	// retrieve new client with applied OAuth2 round tripper
	client = mobileAPI.OAuth2Config.Client(httpClientContext, token)

	mobileAPI.Session.SetClient(client)

	return mobileAPI, nil
}

// mapAPIResponse maps the API response into the passed APIResponse type
func (a *MobileAPI) mapAPIResponse(res *http.Response, apiRes interface{}) (err error) {
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
