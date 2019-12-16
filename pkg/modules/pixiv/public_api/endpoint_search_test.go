package publicapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMobileAPI_GetSearchIllust(t *testing.T) {
	mobileAPI := getTestMobileAPI()

	illustSearchResults, err := mobileAPI.GetSearchIllust(
		"test", SearchModePartialTagMatch, SearchOrderDescending, 1,
	)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(illustSearchResults)
	assert.New(t).NotNil(illustSearchResults.Pagination.Next)
	assert.New(t).Equal(len(illustSearchResults.Illustrations), 1000)

	newIllustSearchResults, err := mobileAPI.GetSearchIllust(
		"test", SearchModePartialTagMatch, SearchOrderDescending, *illustSearchResults.Pagination.Next,
	)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(newIllustSearchResults)
}
