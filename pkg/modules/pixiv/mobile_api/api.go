package mobile_api

import (
	"context"
	watcherHttp "github.com/DaRealFreak/watcher-go/pkg/http"
	"github.com/DaRealFreak/watcher-go/pkg/http/session"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	"golang.org/x/oauth2"
)

// AjaxAPI is the implementation of the API used in the mobile applications
type MobileAPI struct {
	Session      watcherHttp.SessionInterface
	OAuth2Config *oauth2.Config
}

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
