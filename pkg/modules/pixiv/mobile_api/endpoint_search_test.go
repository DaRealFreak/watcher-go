package mobileapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMobileAPI_GetSearchIllust(t *testing.T) {
	illustSearchResults, err := mobileAPI.GetSearchIllust(
		"test", SearchModePartialTagMatch, SearchOrderDateDescending, 0,
	)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(illustSearchResults)
	assert.New(t).Equal(len(illustSearchResults.Illustrations), 30)
	assert.New(t).NotEmpty(illustSearchResults.NextURL)
}

func TestMobileAPI_GetSearchIllustByURL(t *testing.T) {
	illustSearchResults, err := mobileAPI.GetSearchIllust(
		"test", SearchModePartialTagMatch, SearchOrderDateDescending, 0,
	)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(illustSearchResults)
	assert.New(t).Equal(len(illustSearchResults.Illustrations), 30)
	assert.New(t).NotEmpty(illustSearchResults.NextURL)

	nextPageIllustSearchResults, err := mobileAPI.GetSearchIllustByURL(illustSearchResults.NextURL)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(nextPageIllustSearchResults)
	assert.New(t).Equal(len(nextPageIllustSearchResults.Illustrations), 30)
}
