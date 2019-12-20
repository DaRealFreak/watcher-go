package internal

import (
	"golang.org/x/oauth2"
	"strings"
)

func AuthCodeURLImplicit(cfg *oauth2.Config, state string) string {
	return strings.Replace(cfg.AuthCodeURL(state), "response_type=code", "response_type=token", -1)
}
