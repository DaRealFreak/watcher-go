package deviantart

import (
	"testing"

	"github.com/DaRealFreak/watcher-go/pkg/http/session"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

// nolint: gochecknoglobals
var (
	da *deviantArt
)

// TestRetrieveOAuth2CodeTimedOut tests the timout shutdown of the web server
func TestRetrieveOAuth2CodeTimedOut(t *testing.T) {
	assertion := assert.New(t)

	da = &deviantArt{}
	module := models.Module{
		Session:         session.NewSession(),
		LoggedIn:        false,
		ModuleInterface: da,
	}
	da.Module = module

	// wait for timeout returning empty string
	oAuth2Code := da.retrieveOAuth2Code()
	assertion.Equal(oAuth2Code, (*oauth2.Token)(nil))
}
