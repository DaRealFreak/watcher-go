package deviantart

import (
	"golang.org/x/oauth2"
	"net/http"
	"sync"
	"testing"

	"github.com/DaRealFreak/watcher-go/pkg/http/session"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/stretchr/testify/assert"
)

// nolint: gochecknoglobals
var (
	da *deviantArt
)

// TestRetrieveOAuth2CodeCalled tests the return value of the retrieve OAuth2 token if it gets called in time
// FixMe: update for implicit type instead of previous code type
func TestRetrieveOAuth2CodeCalled(t *testing.T) {
	assertion := assert.New(t)

	var wg sync.WaitGroup
	wg.Add(1)
	// routine this check so we can actually call the request
	go checkNoTimeout(t, &wg)
	// call local host resolved domain with test_code as code
	_, err := http.Get("http://lvh.me:8080/da-cb?code=test_code")
	assertion.NoError(err)
	// wait for this test to finish
	wg.Wait()
}

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

// checkNoTimeout checks the web server response if actually called
// to test this we have to goroutine this check and lock it to prevent overlapping port bindings
func checkNoTimeout(t *testing.T, wg *sync.WaitGroup) {
	assertion := assert.New(t)
	ch := make(chan *oauth2.Token)
	// listen with a go routine to be able to time it out
	go func() {
		ch <- da.retrieveOAuth2Code()
	}()

	receivedCode := <-ch
	assertion.Equal(receivedCode, "test_code")
	wg.Done()
}
