package fanboxapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFanboxAPI_GetPostInfo(t *testing.T) {
	postInfo, err := getTestFanboxAPI().GetPostInfo(12345)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(postInfo)
}
