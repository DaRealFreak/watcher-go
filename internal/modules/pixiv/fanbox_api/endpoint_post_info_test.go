package fanboxapi

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestFanboxPostInfo_Unmarshal ensures the post is read from the nested "post" key,
// the API used to return the post directly as the "body" object
func TestFanboxPostInfo_Unmarshal(t *testing.T) {
	response := `{"body":{"post":{"id":"12224285","title":"post title","isRestricted":false,
		"user":{"userId":"13798302","name":"creator name"},"creatorId":"lik","type":"article",
		"coverImageUrl":"https://pixiv.pximg.net/cover.jpeg",
		"body":{"blocks":[
			{"type":"p","text":"see https://example.com/gallery for more"},
			{"type":"image","imageId":"Zhjnzysb6luixUydkoKBLP5U"},
			{"type":"image","imageId":"a8SUHdSidP4RVJhNwZ53wtg2"}
		],"imageMap":{
			"a8SUHdSidP4RVJhNwZ53wtg2":{"id":"a8SUHdSidP4RVJhNwZ53wtg2","extension":"png",
				"originalUrl":"https://downloads.fanbox.cc/images/post/12224285/a8SUHdSidP4RVJhNwZ53wtg2.png"},
			"Zhjnzysb6luixUydkoKBLP5U":{"id":"Zhjnzysb6luixUydkoKBLP5U","extension":"png",
				"originalUrl":"https://downloads.fanbox.cc/images/post/12224285/Zhjnzysb6luixUydkoKBLP5U.png"}
		},"fileMap":{},"embedMap":{},"urlEmbedMap":{}},
		"imageForShare":"https://pixiv.pximg.net/share.jpeg"}}}`

	var postInfo FanboxPostInfo
	assert.New(t).NoError(json.Unmarshal([]byte(response), &postInfo))

	postDetail := postInfo.Body.Post
	assert.New(t).Equal("12224285", postDetail.ID.String())
	assert.New(t).Equal("13798302", postDetail.User.UserId)
	assert.New(t).Equal("https://pixiv.pximg.net/share.jpeg", postDetail.ImageForShare)
	// the blocks reference the images by ID, the order of the blocks defines the download order
	assert.New(t).Equal([]string{
		"https://downloads.fanbox.cc/images/post/12224285/Zhjnzysb6luixUydkoKBLP5U.png",
		"https://downloads.fanbox.cc/images/post/12224285/a8SUHdSidP4RVJhNwZ53wtg2.png",
	}, postDetail.ImagesFromBlocks())
}

// TestFanboxPostInfo_UnmarshalRestricted ensures a post without access, which returns a null
// body, does not fail to unmarshal
func TestFanboxPostInfo_UnmarshalRestricted(t *testing.T) {
	response := `{"body":{"post":{"id":"12224285","title":"post title","isRestricted":true,
		"user":{"userId":"13798302","name":"creator name"},"body":null,"excerpt":"",
		"imageForShare":"https://pixiv.pximg.net/share.jpeg"}}}`

	var postInfo FanboxPostInfo
	assert.New(t).NoError(json.Unmarshal([]byte(response), &postInfo))
	assert.New(t).Equal("12224285", postInfo.Body.Post.ID.String())
	assert.New(t).Empty(postInfo.Body.Post.ImagesFromBlocks())
}

// TestFanboxPostComments_CommentsFromAuthor ensures the comments of the post author are
// collected from the separate post.getComments endpoint, including their replies
func TestFanboxPostComments_CommentsFromAuthor(t *testing.T) {
	response := `{"body":{"viewMode":"LOGIN_REQUIRED","commentList":{"items":[
		{"id":"1","body":"nice work","user":{"userId":"156124","name":"someone"},"replies":[
			{"id":"2","body":"thanks, full set at https://example.com/set","user":{"userId":"13798302","name":"creator name"}}
		]},
		{"id":"3","body":"password is hunter2","user":{"userId":"13798302","name":"creator name"},"replies":[]},
		{"id":"4","body":"unrelated","user":{"userId":"20331264","name":"other"},"replies":[]}
	],"nextUrl":"https://api.fanbox.cc/post.getComments?postId=1&offset=10&limit=10"}}}`

	var comments FanboxPostComments
	assert.New(t).NoError(json.Unmarshal([]byte(response), &comments))
	assert.New(t).Equal([]string{
		"thanks, full set at https://example.com/set",
		"password is hunter2",
	}, comments.CommentsFromAuthor("13798302"))
}

// TestFanboxPostComments_CommentsFromAuthorWithoutAccess ensures a null comment list, which is
// returned if the session may not view the comments, does not panic
func TestFanboxPostComments_CommentsFromAuthorWithoutAccess(t *testing.T) {
	response := `{"body":{"viewMode":"PLEDGE_INSUFFICIENT","commentList":null}}`

	var comments FanboxPostComments
	assert.New(t).NoError(json.Unmarshal([]byte(response), &comments))
	assert.New(t).Empty(comments.CommentsFromAuthor("13798302"))
}

// TestFanboxPostComments_UnmarshalLastPage ensures the last comment page, which returns a null
// nextUrl instead of an empty string, does not fail to unmarshal
func TestFanboxPostComments_UnmarshalLastPage(t *testing.T) {
	response := `{"body":{"viewMode":"OPEN","commentList":{"items":[
		{"id":"12410745","parentCommentId":"0","rootCommentId":"0","body":"Oh my god.",
			"createdDatetime":"2026-07-08T09:34:51+09:00","likeCount":0,"isLiked":false,"isOwn":false,
			"user":{"userId":"104696042","name":"NeoArc00","iconUrl":"https://pixiv.pximg.net/icon.jpeg"},"replies":[]},
		{"id":"12403579","parentCommentId":"0","rootCommentId":"0","body":"Amazing",
			"createdDatetime":"2026-07-06T20:20:00+09:00","likeCount":1,"isLiked":false,"isOwn":false,
			"user":{"userId":"73225384","name":"SpaceMan","iconUrl":null},"replies":[]}
	],"nextUrl":null}}}`

	var comments FanboxPostComments
	assert.New(t).NoError(json.Unmarshal([]byte(response), &comments))
	assert.New(t).Len(comments.Body.CommentList.Items, 2)
	assert.New(t).Empty(comments.Body.CommentList.NextURL)
	// none of the comments are from the creator of the post
	assert.New(t).Empty(comments.CommentsFromAuthor("13798302"))
}

func TestFanboxAPI_GetPostInfo(t *testing.T) {
	postInfo, err := getTestFanboxAPI().GetPostInfo(12345)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(postInfo)
}
