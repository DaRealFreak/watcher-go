package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	watcherHttp "github.com/DaRealFreak/watcher-go/internal/http"
	"github.com/DaRealFreak/watcher-go/internal/models"
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

// loginData is the json struct for the JSON login form
type loginFormData struct {
	Username string `json:"login"`
	Password string `json:"password"`
}

// loginResponse is the login response during the authentication process
type loginResponse struct {
	Success      bool   `json:"success"`
	TokenType    string `json:"token_type"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// NewSankakuComplexApi returns the settings of the SankakuComplex API
func NewSankakuComplexApi(moduleKey string, session watcherHttp.SessionInterface, account *models.Account) *SankakuComplexApi {
	var response loginResponse

	if account != nil {
		// initial request to check for success of login and to retrieve the access/refresh token
		loginData := loginFormData{}
		loginData.Username = account.Username
		loginData.Password = account.Password

		data, _ := json.Marshal(loginData)
		req, _ := http.NewRequest(
			"POST",
			"https://capi-v2.sankakucomplex.com/auth/token",
			bytes.NewReader(data),
		)

		req.Header.Set("Accept", "application/vnd.sankaku.api+json;v=2")
		req.Header.Set("Content-Type", "application/json;charset=utf-8")

		res, err := session.GetClient().Do(req)
		if err != nil {
			return nil
		}

		body, _ := ioutil.ReadAll(res.Body)
		_ = json.Unmarshal(body, &response)
	}

	oauthConfig := oauth2.Config{
		Endpoint: oauth2.Endpoint{
			AuthURL: "https://capi-v2.sankakucomplex.com/auth/token",
		},
	}

	ctx := context.Background()
	api := &SankakuComplexApi{
		Session: session,
		tokenSrc: oauthConfig.TokenSource(ctx, &oauth2.Token{
			AccessToken:  response.AccessToken,
			TokenType:    response.TokenType,
			RefreshToken: response.RefreshToken,
		}),
		account:     account,
		rateLimiter: rate.NewLimiter(rate.Every(1*time.Millisecond), 1),
		ctx:         ctx,
		moduleKey:   moduleKey,
	}

	// if the login was successful add our round tripper to add the authorization header on requests
	if response.Success {
		client := session.GetClient()
		client.Transport = api.addRoundTripper(client.Transport)
	}

	return api
}

func (a *SankakuComplexApi) LoginSuccessful() bool {
	tk, err := a.tokenSrc.Token()
	if err != nil {
		return false
	}

	return tk.AccessToken != "" && tk.Valid()
}
