// Package ajaxapi handles the AJAX functionality which is not usable from neither the public or the mobile API
// such as the fanboxes or possibly the like functionality
package ajaxapi

import (
	"encoding/json"
	"net/http"
	"net/url"

	watcherHttp "github.com/DaRealFreak/watcher-go/pkg/http"
	"github.com/DaRealFreak/watcher-go/pkg/http/session"
)

// AjaxAPI is the implementation of the not reachable but required endpoints not in the public or mobile API
type AjaxAPI struct {
	StorageURL *url.URL
	Session    watcherHttp.SessionInterface
	LoginData  LoginData
}

// NewAjaxAPI initializes the module
func NewAjaxAPI(moduleKey string) *AjaxAPI {
	ajaxAPI := &AjaxAPI{
		Session: session.NewSession(moduleKey),
	}
	ajaxAPI.StorageURL, _ = url.Parse("https://pixiv.net")

	return ajaxAPI
}

// SetCookies sets the required pixiv session cookies required for the AJAX API
func (a *AjaxAPI) SetCookies() {
	a.Session.GetClient().Jar.SetCookies(a.StorageURL,
		[]*http.Cookie{
			{Name: "device_token", Value: a.LoginData.DeviceToken},
			{Name: "PHPSESSID", Value: a.LoginData.SessionID},
		},
	)
}

// SetPixivRoundTripper adds a round tripper to add the required and checked request headers on every sent request
func (a *AjaxAPI) SetPixivRoundTripper() {
	client := a.Session.GetClient()
	client.Transport = SetPixivWebHeaders(client.Transport, a.LoginData)
}

// mapAPIResponse maps the API response into the passed APIResponse type
func (a *AjaxAPI) mapAPIResponse(res *http.Response, apiRes interface{}) (err error) {
	content := a.Session.GetDocument(res).Text()

	if res.StatusCode >= 400 {
		var apiErr APIError
		// unmarshal the request content into the error struct
		if err := json.Unmarshal([]byte(content), &apiErr); err != nil {
			return err
		}

		return apiErr
	}

	// unmarshal the request content into the response struct
	if err := json.Unmarshal([]byte(content), &apiRes); err != nil {
		return err
	}

	return nil
}
