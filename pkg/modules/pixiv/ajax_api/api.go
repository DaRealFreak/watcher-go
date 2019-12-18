// Package ajaxapi handles the AJAX functionality which is not usable from neither the public or the mobile API
// such as the fanboxes or possibly the like functionality
package ajaxapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"

	watcherHttp "github.com/DaRealFreak/watcher-go/pkg/http"
	"github.com/DaRealFreak/watcher-go/pkg/http/session"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	log "github.com/sirupsen/logrus"
)

// CookieSession is the session cookie which is set after a successful login
const CookieSession = "PHPSESSID"

// AjaxAPI is the implementation of the not reachable but required endpoints not in the public or mobile API
type AjaxAPI struct {
	StorageURL    *url.URL
	Session       watcherHttp.SessionInterface
	Key           string
	SessionCookie *models.Cookie
}

// NewAjaxAPI initializes the AJAX API and handles the whole auth and round tripper procedures
func NewAjaxAPI(moduleKey string) *AjaxAPI {
	ajaxAPI := &AjaxAPI{
		Key:     moduleKey,
		Session: session.NewSession(moduleKey),
	}
	ajaxAPI.StorageURL, _ = url.Parse("https://pixiv.net")

	return ajaxAPI
}

// AddRoundTrippers sets the required pixiv session sessionCookie required for the AJAX API
// and adds the round trippers to the session client
func (a *AjaxAPI) AddRoundTrippers() {
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
func (a *AjaxAPI) setPixivRoundTripper() {
	client := a.Session.GetClient()
	client.Transport = a.setPixivWebHeaders(client.Transport, a.SessionCookie)
}

// mapAPIResponse maps the API response into the passed APIResponse type
func (a *AjaxAPI) mapAPIResponse(res *http.Response, apiRes interface{}) (err error) {
	content := a.Session.GetDocument(res).Text()

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
