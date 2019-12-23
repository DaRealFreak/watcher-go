package publicapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMobileAPI_GetSearchIllust(t *testing.T) {
	illustSearchResults, err := publicAPI.GetSearchIllust(
		"test", SearchModePartialTagMatch, SearchOrderDescending, 1,
	)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(illustSearchResults)
	assert.New(t).NotNil(illustSearchResults.Pagination.Next)
	assert.New(t).Equal(len(illustSearchResults.Illustrations), 1000)

	newIllustSearchResults, err := publicAPI.GetSearchIllust(
		"test", SearchModePartialTagMatch, SearchOrderDescending, *illustSearchResults.Pagination.Next,
	)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(newIllustSearchResults)
}
