// Package fanboxapi handles the fanbox functionality which is not usable from the mobile API
package fanboxapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	watcherHttp "github.com/DaRealFreak/watcher-go/internal/http"
	"github.com/DaRealFreak/watcher-go/internal/http/session"
	"github.com/DaRealFreak/watcher-go/internal/models"
	log "github.com/sirupsen/logrus"
)

// CookieSession is the session cookie which is set after a successful login
const CookieSession = "FANBOXSESSID"

// FanboxAPI is the implementation of the not reachable but required endpoints not in the public or mobile API
type FanboxAPI struct {
	StorageURL    *url.URL
	Session       watcherHttp.SessionInterface
	Key           string
	SessionCookie *models.Cookie
}

// NewFanboxAPI initializes the Fanbox API and handles the whole auth and round tripper procedures
func NewFanboxAPI(moduleKey string) *FanboxAPI {
	fanboxAPI := &FanboxAPI{
		Key:     moduleKey,
		Session: session.NewSession(moduleKey),
	}
	fanboxAPI.StorageURL, _ = url.Parse("https://www.fanbox.cc")

	return fanboxAPI
}

// AddRoundTrippers sets the required pixiv session sessionCookie required for the Fanbox API
// and adds the round trippers to the session client
func (a *FanboxAPI) AddRoundTrippers() {
	if a.SessionCookie == nil {
		log.WithField("module", a.Key).Fatalf(
			"required cookie %s does not exist or expired", CookieSession,
		)
		os.Exit(1)
	}

	a.Session.GetClient().Jar.SetCookies(a.StorageURL,
		[]*http.Cookie{
			{Name: CookieSession, Value: a.SessionCookie.Value},
		},
	)

	// also apply sessionCookie for our cookie header in the round trip
	a.setPixivRoundTripper()
}

// setPixivRoundTripper adds a round tripper to add the required and checked request headers on every sent request
func (a *FanboxAPI) setPixivRoundTripper() {
	client := a.Session.GetClient()
	client.Transport = a.setPixivWebHeaders(client.Transport, a.SessionCookie)
}

// mapAPIResponse maps the API response into the passed APIResponse type
func (a *FanboxAPI) mapAPIResponse(res *http.Response, apiRes interface{}) (err error) {
	out, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	content := string(out)

	if res.StatusCode >= 400 {
		var (
			apiErr    APIError
			apiReqErr APIRequestError
		)

		if err := json.Unmarshal([]byte(content), &apiErr); err == nil {
			return apiErr
		}

		if err := json.Unmarshal([]byte(content), &apiReqErr); err == nil {
			return &apiReqErr
		}

		return fmt.Errorf(`unknown error response: "%s"`, content)
	}

	// unmarshal the request content into the response struct
	if err := json.Unmarshal([]byte(content), &apiRes); err != nil {
		return err
	}

	return nil
}
