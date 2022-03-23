package models

import (
	"context"
	"net/http"
	"time"

	"golang.org/x/oauth2"
)

// OAuthClient contains all the required details to create an OAuth client aside from URLs
type OAuthClient struct {
	ID           int
	ClientID     string
	ClientSecret string
	AccessToken  string
	RefreshToken string
	Module       string
	Disabled     bool
}

// GetClient returns a http client implementing the OAuth2 configuration
func (c *OAuthClient) GetClient(authURL string, tokenURL string, scopes []string) *http.Client {
	config := oauth2.Config{
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  authURL,
			TokenURL: tokenURL,
		},
		Scopes: scopes,
	}

	token := oauth2.Token{
		AccessToken:  c.AccessToken,
		RefreshToken: c.RefreshToken,
		// Must be non-nil, otherwise token will not be expired
		Expiry: time.Now().Add(-24 * time.Hour),
	}

	return config.Client(context.Background(), &token)
}
