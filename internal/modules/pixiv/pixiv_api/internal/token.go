// Package internal contains all the internal functions for the pixiv API token retrieval and refresh process
package internal

import (
	"encoding/json"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/oauth2"
)

// tokenResponse is the OAuth2 Token response from the Pixiv OAuth2 Token URL
type tokenResponse struct {
	Response struct {
		AccessToken  string      `json:"access_token"`
		ExpiresIn    json.Number `json:"expires_in"`
		TokenType    string      `json:"token_type"`
		Scope        string      `json:"scope"`
		RefreshToken string      `json:"refresh_token"`
		DeviceToken  string      `json:"device_token"`
	} `json:"response"`
}

// PasswordCredentialsToken is a custom implementation of the oauth2 PasswordCredentialsToken since additional
// post values are required and checked server side from pixiv
func PasswordCredentialsToken(
	account *models.OAuthClient, cfg *oauth2.Config, client *http.Client,
) (*oauth2.Token, error) {
	v := url.Values{
		"get_secure_url": {"1"},
		"client_id":      {cfg.ClientID},
		"client_secret":  {cfg.ClientSecret},
		// password specific values
		"grant_type":    {"refresh_token"},
		"refresh_token": {account.RefreshToken},
	}

	res, err := client.PostForm(cfg.Endpoint.TokenURL, v)
	if err != nil {
		return nil, err
	}

	return RetrieveTokenFromResponse(res)
}

// RetrieveTokenFromResponse extracts the OAuth2 Token from the passed http Response
func RetrieveTokenFromResponse(response *http.Response) (*oauth2.Token, error) {
	var token tokenResponse

	bytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(bytes, &token); err != nil {
		return nil, err
	}

	tokenExpiry, err := token.Response.ExpiresIn.Int64()
	if err != nil {
		return nil, err
	}

	return &oauth2.Token{
		AccessToken:  token.Response.AccessToken,
		TokenType:    token.Response.TokenType,
		RefreshToken: token.Response.RefreshToken,
		Expiry:       time.Now().Add(time.Duration(tokenExpiry) * time.Second),
	}, nil
}
