// Package api is the implementation of the DeviantArt API including the authentication using the Implicit Grant OAuth2
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	cloudflarebp "github.com/DaRealFreak/cloudflare-bp-go"
	watcherHttp "github.com/DaRealFreak/watcher-go/internal/http"
	"github.com/DaRealFreak/watcher-go/internal/http/session"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	implicitoauth2 "github.com/DaRealFreak/watcher-go/pkg/oauth2"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"golang.org/x/time/rate"
)

// MaxDeviationsPerPage is the maximum amount of results you can retrieve from the API in one request
const MaxDeviationsPerPage = 24

// DeviantartAPI contains all required items to communicate with the API
type DeviantartAPI struct {
	Session           watcherHttp.SessionInterface
	rateLimiter       *rate.Limiter
	ctx               context.Context
	OAuth2Config      *oauth2.Config
	account           *models.Account
	useConsoleExploit bool
	moduleKey         string
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
		rateLimiter: rate.NewLimiter(rate.Every(10*time.Second), 1),
		ctx:         context.Background(),
		moduleKey:   moduleKey,
	}
}

// AddRoundTrippers adds the round trippers for CloudFlare, adds a custom user agent
// and implements the implicit OAuth2 authentication and sets the Token round tripper
func (a *DeviantartAPI) AddRoundTrippers() {
	client := a.Session.GetClient()
	// apply CloudFlare bypass
	client.Transport = cloudflarebp.AddCloudFlareByPass(client.Transport)
	jar := client.Jar

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
	a.applyRateLimit()

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
		if a.useConsoleExploit {
			log.WithField("module", a.moduleKey).Info(
				"reached console exploit request limit too. sleeping 5 minutes to let account limits recover",
			)
			time.Sleep(5 * time.Minute)
		}

		// toggle console exploit to relieve the other API method
		a.useConsoleExploit = !a.useConsoleExploit

		return a.request(method, endpoint, values)
	}

	return res, err
}

// mapAPIResponse maps the API response into the passed APIResponse type
func (a *DeviantartAPI) mapAPIResponse(res *http.Response, apiRes interface{}) (err error) {
	out, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	content := string(out)

	if res.StatusCode >= 400 {
		var apiErr Error

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
func (a *DeviantartAPI) applyRateLimit() {
	raven.CheckError(a.rateLimiter.Wait(a.ctx))
}
