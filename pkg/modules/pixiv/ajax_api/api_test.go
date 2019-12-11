package ajaxapi

import (
	"fmt"
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

	creatorInfo, err := ajaxAPI.GetCreator(12345)
	if err != nil {
		panic(err)
	}

	postInfo, err := ajaxAPI.GetPostList(12345)
	if err != nil {
		panic(err)
	}

	nextPagePostInfo, err := ajaxAPI.GetPostListByURL(creatorInfo.Body.Post.NextURL)
	if err != nil {
		panic(err)
	}

	fmt.Println(postInfo)
	fmt.Println(nextPagePostInfo)
}
