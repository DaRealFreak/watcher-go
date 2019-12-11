package ajaxapi

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGetFiles tests if an archive can retrieve all file names/paths from the generated archive
func TestLogin(t *testing.T) {
	ajaxAPI := NewAjaxAPI("pixiv AJAX API")
	ajaxAPI.Cookies.SessionID = os.Getenv("PIXIV_SESSION_ID")
	ajaxAPI.Cookies.DeviceToken = os.Getenv("PIXIV_DEVICE_TOKEN")

	ajaxAPI.SetCookies()
	ajaxAPI.SetPixivRoundTripper()

	creatorInfo, err := ajaxAPI.GetCreator(12345)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(creatorInfo)

	nextPagePostInfo, err := ajaxAPI.GetPostListByURL(creatorInfo.Body.Post.NextURL)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(nextPagePostInfo)

	postList, err := ajaxAPI.GetPostList(12345)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(postList)

	if len(postList.Body.Items) > 0 {
		postID, _ := postList.Body.Items[0].ID.Int64()
		postInfo, err := ajaxAPI.GetPostInfo(int(postID))
		assert.New(t).NoError(err)
		assert.New(t).NotNil(postInfo)
	}
}
