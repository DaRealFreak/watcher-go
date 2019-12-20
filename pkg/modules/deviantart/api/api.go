// Package api is the implementation of the DeviantArt API including the authentication using the Implicit Grant OAuth2
package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
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

	a.Session.SetClient(
		oauth2.NewClient(
			context.Background(),
			&implicitoauth2.ImplicitGrantTokenSource{
				Grant: NewImplicitGrantDeviantart(a.OAuth2Config, client, a.account),
			},
		),
	)
}

// Request simulates the http.NewRequest method to add the additional option
// to use the DiFi API console exploit to circumvent API limitations
func (a *DeviantartAPI) Request(method string, endpoint string, values url.Values) (*http.Response, error) {
	apiRequestURL := "https://www.deviantart.com/api/v1/oauth2" + endpoint

	switch strings.ToUpper(method) {
	case "GET":
		requestURL, err := url.Parse(apiRequestURL)
		if err != nil {
			return nil, err
		}

		existingValues := requestURL.Query()

		for key, group := range values {
			for _, value := range group {
				existingValues.Add(key, value)
			}
		}

		requestURL.RawQuery = existingValues.Encode()

		return a.Session.Get(requestURL.String())
	case "POST":
		return a.Session.Post(endpoint, values)
	default:
		return nil, fmt.Errorf("unknown request method: %s", method)
	}
}
