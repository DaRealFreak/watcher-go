package ajaxapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAjaxAPI_GetCreator(t *testing.T) {
	creatorInfo, err := getTestAjaxAPI().GetCreator(8189060)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(creatorInfo)
}

func TestAjaxAPI_GetPostList(t *testing.T) {
	postList, err := getTestAjaxAPI().GetPostList(8189060, 50)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(postList)
}

func TestAjaxAPI_GetPostListByURL(t *testing.T) {
	// retrieve next URL from previous Post List (we are using 200 items as limit, we need a creator with > 200 items)
	postList, err := getTestAjaxAPI().GetPostList(8189060, 20)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(postList)
	assert.New(t).Equal(len(postList.Body.Items), 20)
	assert.New(t).NotEmpty(postList.Body.NextURL)

	nextPagePostList, err := getTestAjaxAPI().GetPostListByURL(postList.Body.NextURL)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(nextPagePostList)
	assert.New(t).Equal(len(nextPagePostList.Body.Items), 20)

	// retrieve next URL from creator list
	creatorInfo, err := getTestAjaxAPI().GetCreator(8189060)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(creatorInfo)
	assert.New(t).NotEmpty(creatorInfo.Body.Post.NextURL)

	nextPagePostList, err = getTestAjaxAPI().GetPostListByURL(creatorInfo.Body.Post.NextURL)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(nextPagePostList)
}
