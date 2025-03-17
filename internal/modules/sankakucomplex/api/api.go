package api

import (
	"context"
	"time"

	watcherHttp "github.com/DaRealFreak/watcher-go/internal/http"
	"github.com/DaRealFreak/watcher-go/internal/models"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"golang.org/x/time/rate"
)

// SankakuComplexApi contains all required items to communicate with the API
type SankakuComplexApi struct {
	Session     watcherHttp.SessionInterface
	tokenSrc    oauth2.TokenSource
	rateLimiter *rate.Limiter
	ctx         context.Context
	account     *models.Account
	moduleKey   string
}

// NewSankakuComplexApi returns the settings of the SankakuComplex API.
// It now uses the new OIDC flow (implemented in the iodc package) to retrieve an OAuth token.
func NewSankakuComplexApi(moduleKey string, session watcherHttp.SessionInterface, account *models.Account) *SankakuComplexApi {
	ctx := context.Background()

	// OIDC configuration.
	oidcCfg := Config{
		InitialAuthURL: "https://login.sankakucomplex.com/oidc/auth?client_id=sankaku-web-app&scope=openid&response_type=code&route=login&theme=black&lang=en&redirect_uri=https%3A%2F%2Fwww.sankakucomplex.com%2Fsso%2Fcallback&state=return_uri%3Dhttps%3A%2F%2Fwww.sankakucomplex.com%2F&is_iframe=true&is_mobile=false",
		TokenURL:       "https://login.sankakucomplex.com/auth/token",
		FinalizeURL:    "https://sankakuapi.com/sso/finalize?lang=en",
		ClientID:       "sankaku-web-app",
		RedirectURI:    "https://www.sankakucomplex.com/sso/callback",
	}

	// Create an OIDC client.
	oidc, err := NewOIDCClient(oidcCfg)
	if err != nil {
		log.WithField("module", moduleKey).Warnf("failed to create OIDC client: %s", err.Error())
		return nil
	}

	// Use the OIDC flow to get an OAuth token.
	token, err := oidc.GetOAuthToken(ctx, account.Username, account.Password)
	if err != nil {
		log.WithField("module", moduleKey).Warnf("OIDC authentication failed: %s", err.Error())
		return nil
	}

	// Create an oauth2.TokenSource using the retrieved token.
	tokenSrc := oauth2.StaticTokenSource(token)

	api := &SankakuComplexApi{
		Session:     session,
		tokenSrc:    tokenSrc,
		account:     account,
		rateLimiter: rate.NewLimiter(rate.Every(1*time.Millisecond), 1),
		ctx:         ctx,
		moduleKey:   moduleKey,
	}

	// If the token is valid, add a round tripper that injects the Authorization header.
	if token.AccessToken != "" && token.Valid() {
		client := session.GetClient()
		client.Transport = api.addRoundTripper(client.Transport)
	}

	return api
}

// LoginSuccessful checks if a valid access token exists.
func (a *SankakuComplexApi) LoginSuccessful() bool {
	tk, err := a.tokenSrc.Token()
	if err != nil {
		return false
	}
	return tk.AccessToken != "" && tk.Valid()
}
