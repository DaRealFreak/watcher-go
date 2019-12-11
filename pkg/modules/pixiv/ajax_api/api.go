// Package ajax_api handles the AJAX functionality which is not usable from neither the public or the mobile API
// such as the fanboxes or possibly the like functionality
package ajax_api

import (
	"net/http"
	"net/url"

	watcherHttp "github.com/DaRealFreak/watcher-go/pkg/http"
	"github.com/DaRealFreak/watcher-go/pkg/http/session"
)

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

func (a *AjaxAPI) SetPixivRoundTripper() {
	client := a.Session.GetClient()
	client.Transport = SetPixivWebHeaders(client.Transport, a.LoginData)
}
