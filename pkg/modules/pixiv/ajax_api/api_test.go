package ajaxapi

import (
	"os"
	"testing"
)

// getTestAjaxAPI builds the AJAX API from the environment variables and executes all required functions
func getTestAjaxAPI() *AjaxAPI {
	ajaxAPI := NewAjaxAPI("pixiv AJAX API")
	ajaxAPI.Cookies.SessionID = os.Getenv("PIXIV_SESSION_ID")
	ajaxAPI.Cookies.DeviceToken = os.Getenv("PIXIV_DEVICE_TOKEN")

	ajaxAPI.SetCookies()
	ajaxAPI.SetPixivRoundTripper()

	return ajaxAPI
}

// TestLogin tests
func TestLogin(t *testing.T) {
	// ToDo: check if cookies are correct
}
