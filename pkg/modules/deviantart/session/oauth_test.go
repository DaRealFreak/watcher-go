package session

import (
	"testing"

	"github.com/DaRealFreak/watcher-go/pkg/http/session"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

// TestRetrieveOAuth2TokenTimedOut tests the timout of the OAuth2 Token retrieval
func TestRetrieveOAuth2TokenTimedOut(t *testing.T) {
	assertion := assert.New(t)

	da := &DeviantArtSession{
		DefaultSession: session.NewSession(nil),
		TokenStore:     NewTokenStore(),
	}

	// wait for timeout returning empty string
	oAuth2Code := da.retrieveOAuth2Token("basic")
	assertion.Equal(oAuth2Code, (*oauth2.Token)(nil))
}
