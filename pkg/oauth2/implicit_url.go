package implicitoauth2

import (
	"strings"

	"golang.org/x/oauth2"
)

// AuthTokenURL is a function to retrieve the URL to request the Token with by the Implicit Grant OAuth2 authentication
func AuthTokenURL(cfg *oauth2.Config, state string) string {
	return strings.ReplaceAll(cfg.AuthCodeURL(state), "response_type=code", "response_type=token")
}
