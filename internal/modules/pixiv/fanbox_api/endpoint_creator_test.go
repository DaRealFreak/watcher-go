package fanboxapi

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestPostPagination_Unmarshal ensures the paginated page URLs are read from the nested
// "pageUrls" key, the API used to return the URLs directly as the "body" array
func TestPostPagination_Unmarshal(t *testing.T) {
	response := `{"body":{"pageUrls":[
		"https://api.fanbox.cc/post.listCreator?creatorId=lik&firstPublishedDatetime=2026-07-09%2019%3A18%3A13&firstId=12224285&sort=newest&limit=10",
		"https://api.fanbox.cc/post.listCreator?creatorId=lik&firstPublishedDatetime=2026-04-27%2022%3A19%3A27&firstId=11808168&sort=newest&limit=10"
	]}}`

	var pagination PostPagination
	assert.New(t).NoError(json.Unmarshal([]byte(response), &pagination))
	assert.New(t).Len(pagination.Body.URLs, 2)
	assert.New(t).Contains(pagination.Body.URLs[0], "firstId=12224285")
}

// TestPostInfoSinglePage_Unmarshal ensures the posts are read from the nested "posts" key,
// the API used to return the posts directly as the "body" array
func TestPostInfoSinglePage_Unmarshal(t *testing.T) {
	response := `{"body":{"posts":[
		{"id":"12259357","title":"post title","user":{"userId":"8189060","name":"creator name"}},
		{"id":"12241255","title":"other title","user":{"userId":"8189060","name":"creator name"}}
	]}}`

	var postList PostInfoSinglePage
	assert.New(t).NoError(json.Unmarshal([]byte(response), &postList))
	assert.New(t).Len(postList.Body.Posts, 2)
	assert.New(t).Equal("12259357", postList.Body.Posts[0].ID.String())
	assert.New(t).Equal("8189060/creator name", postList.Body.Posts[0].User.GetUserTag())
}

func TestFanboxAPI_GetCreator(t *testing.T) {
	creatorInfo, err := getTestFanboxAPI().GetCreator("mito-nagishiro")
	assert.New(t).NoError(err)
	assert.New(t).NotNil(creatorInfo)
}

func TestFanboxAPI_GetPostList(t *testing.T) {
	postList, err := getTestFanboxAPI().GetPostList("mito-nagishiro", nil, 0, 50)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(postList)
}

func TestFanboxAPI_GetPostListByURL(t *testing.T) {
	// retrieve next URL from previous Post List (user requires to have >= 40 fanbox posts for unit tests to pass)
	postList, err := getTestFanboxAPI().GetPostList("mito-nagishiro", nil, 0, 20)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(postList)
}
