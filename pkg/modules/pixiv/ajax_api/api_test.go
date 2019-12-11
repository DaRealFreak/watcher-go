package ajax_api

import (
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

// TestGetFiles tests if an archive can retrieve all file names/paths from the generated archive
func TestLogin(t *testing.T) {
	ajaxAPI := NewAjaxAPI("pixiv AJAX API")
	ajaxAPI.LoginData.SessionID = os.Getenv("PIXIV_SESSION_ID")
	ajaxAPI.LoginData.DeviceToken = os.Getenv("PIXIV_DEVICE_TOKEN")

	ajaxAPI.SetCookies()
	ajaxAPI.SetPixivRoundTripper()

	// https://www.pixiv.net/ajax/fanbox/creator?userId=12345
	res, err := ajaxAPI.Session.Get("https://fanbox.pixiv.net/api/post.info?postId=12345")
	if err != nil {
		panic(err)
	}

	out, _ := gzip.NewReader(res.Body)
	outText, _ := ioutil.ReadAll(out)
	fmt.Println(string(outText))
}
