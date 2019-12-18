package pixivapi

import (
	"encoding/json"
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

// passwordCredentialsToken is a custom implementation of the oauth2 PasswordCredentialsToken since additional
// post values are required and checked server side from pixiv
func (a *PixivAPI) passwordCredentialsToken(username string, password string) (*oauth2.Token, error) {
	v := url.Values{
		"device_token":   {"pixiv"},
		"get_secure_url": {"true"},
		"include_policy": {"true"},
		"client_id":      {a.OAuth2Config.ClientID},
		"client_secret":  {a.OAuth2Config.ClientSecret},
		// password specific values
		"grant_type": {"password"},
		"username":   {username},
		"password":   {password},
	}

	res, err := a.Session.Post(a.OAuth2Config.Endpoint.TokenURL, v)
	if err != nil {
		return nil, err
	}

	return a.retrieveTokenFromResponse(res)
}

// retrieveTokenFromResponse extracts the OAuth2 Token from the passed http Response
func (a *PixivAPI) retrieveTokenFromResponse(response *http.Response) (*oauth2.Token, error) {
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
