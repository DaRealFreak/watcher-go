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
	Session           watcherHttp.SessionInterface
	rateLimiter       *rate.Limiter
	ctx               context.Context
	OAuth2Config      *oauth2.Config
	account           *models.Account
	useConsoleExploit bool
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
	jar := client.Jar

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

	// OAuth2.NewClient creates completely new client and throws away our cookie jar, set it again
	a.Session.GetClient().Jar = jar
}

// request simulates the http.NewRequest method to add the additional option
// to use the DiFi API console exploit to circumvent API limitations
func (a *DeviantartAPI) request(method string, endpoint string, values url.Values) (res *http.Response, err error) {
	if a.useConsoleExploit {
		res, err = a.consoleRequest(endpoint, values)
	} else {
		apiRequestURL := fmt.Sprintf("https://www.deviantart.com/api/v1/oauth2%s", endpoint)

		switch strings.ToUpper(method) {
		case "GET":
			requestURL, _ := url.Parse(apiRequestURL)
			existingValues := requestURL.Query()

			for key, group := range values {
				for _, value := range group {
					existingValues.Add(key, value)
				}
			}

			requestURL.RawQuery = existingValues.Encode()

			res, err = a.Session.Get(requestURL.String())
		case "POST":
			res, err = a.Session.Post(apiRequestURL, values)
		default:
			return nil, fmt.Errorf("unknown request method: %s", method)
		}
	}

	// rate limitation
	if res != nil && res.StatusCode == 429 {
		// toggle console exploit to relieve the other API method
		a.useConsoleExploit = !a.useConsoleExploit

		return a.request(method, endpoint, values)
	}

	return res, err
}
