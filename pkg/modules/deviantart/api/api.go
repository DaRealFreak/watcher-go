// Package api is the implementation of the DeviantArt API including the authentication using the Implicit Grant OAuth2
package api

import (
	"context"
	"fmt"
	"time"

	watcherHttp "github.com/DaRealFreak/watcher-go/pkg/http"
	"github.com/DaRealFreak/watcher-go/pkg/http/session"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	implicitoauth2 "github.com/DaRealFreak/watcher-go/pkg/oauth2"
	browser "github.com/EDDYCJY/fake-useragent"
	"golang.org/x/oauth2"
	"golang.org/x/time/rate"
)

// DeviantartAPI contains all required items to communicate with the API
type DeviantartAPI struct {
	Session      watcherHttp.SessionInterface
	rateLimiter  *rate.Limiter
	ctx          context.Context
	OAuth2Config *oauth2.Config
	account      *models.Account
}

// NewDeviantartAPI returns the settings of the DeviantArt API
func NewDeviantartAPI(moduleKey string, account *models.Account) *DeviantartAPI {
	return &DeviantartAPI{
		Session: session.NewSession(moduleKey),
		OAuth2Config: &oauth2.Config{
			ClientID: "9991",
			Endpoint: oauth2.Endpoint{
				AuthURL: "https://www.deviantart.com/oauth2/authorize",
			},
			Scopes:      []string{"basic", "browse", "gallery", "feed"},
			RedirectURL: "https://lvh.me/da-cb",
		},
		account:     account,
		rateLimiter: rate.NewLimiter(rate.Every(1500*time.Millisecond), 1),
		ctx:         context.Background(),
	}
}

// AddRoundTrippers adds the round trippers for CloudFlare, adds a custom user agent
// and implements the implicit OAuth2 authentication and sets the Token round tripper
func (a *DeviantartAPI) AddRoundTrippers() {
	client := a.Session.GetClient()

	client.Transport = a.SetCloudFlareHeaders(client.Transport)
	client.Transport = a.SetUserAgent(client.Transport, browser.Firefox())

	grant := NewImplicitGrantDeviantart(a.OAuth2Config, client, a.account)

	token, err := grant.Token()
	fmt.Println(token, err)

	fmt.Println(implicitoauth2.AuthTokenURL(a.OAuth2Config, "session-id"))
}
