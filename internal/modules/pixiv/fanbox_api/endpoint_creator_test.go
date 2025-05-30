package fanboxapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
