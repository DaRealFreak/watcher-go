package ajaxapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAjaxAPI_GetPostInfo(t *testing.T) {
	postInfo, err := getTestAjaxAPI().GetPostInfo(12345)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(postInfo)
}
