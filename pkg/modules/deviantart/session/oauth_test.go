package session

import (
	"testing"

	"github.com/DaRealFreak/watcher-go/pkg/http/session"
	browser "github.com/EDDYCJY/fake-useragent"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

// TestRetrieveOAuth2TokenTimedOut tests the timout of the OAuth2 Token retrieval
func TestRetrieveOAuth2TokenTimedOut(t *testing.T) {
	assertion := assert.New(t)

	da := &DeviantArtSession{
		DefaultSession: session.NewSession(t.Name()),
		TokenStore:     NewTokenStore(),
		DefaultHeaders: map[string]string{
			"User-Agent":      browser.Chrome(),
			"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
			"Accept-Encoding": "gzip, deflate, br",
			"Accept-Language": "en-US;en;q=0.5",
		},
	}

	// wait for timeout returning empty string
	oAuth2Code := da.retrieveOAuth2Token("basic")
	assertion.Equal(oAuth2Code, (*oauth2.Token)(nil))
}
