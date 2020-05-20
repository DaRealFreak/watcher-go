package ajaxapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAjaxAPI_GetCreator(t *testing.T) {
	creatorInfo, err := getTestAjaxAPI().GetCreator("mito-nagishiro")
	assert.New(t).NoError(err)
	assert.New(t).NotNil(creatorInfo)
}

func TestAjaxAPI_GetPostList(t *testing.T) {
	postList, err := getTestAjaxAPI().GetPostList("mito-nagishiro", 50)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(postList)
}

func TestAjaxAPI_GetPostListByURL(t *testing.T) {
	// retrieve next URL from previous Post List (user requires to have >= 40 fanbox posts for unit tests to pass)
	postList, err := getTestAjaxAPI().GetPostList("mito-nagishiro", 20)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(postList)
	assert.New(t).Equal(len(postList.Body.Items), 20)
	assert.New(t).NotEmpty(postList.Body.NextURL)

	nextPagePostList, err := getTestAjaxAPI().GetPostListByURL(postList.Body.NextURL)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(nextPagePostList)
	assert.New(t).Equal(len(nextPagePostList.Body.Items), 20)
}
