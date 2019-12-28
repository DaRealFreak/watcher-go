package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeviantartAPI_GalleryAll(t *testing.T) {
	daAPI.useConsoleExploit = false

	collection, err := daAPI.GalleryAll(
		"CLG-Artisa", 0, MaxDeviationsPerPage,
	)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(collection)

	// toggle console exploit, we also require the first OAuth2 process to have succeeded
	// since we require the user information cookie which is set on a successful login
	daAPI.useConsoleExploit = true

	tagResultsConsoleExploit, err := daAPI.GalleryAll(
		"CLG-Artisa", 0, MaxDeviationsPerPage,
	)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(tagResultsConsoleExploit)

	assert.New(t).Equal(MaxDeviationsPerPage, len(collection.Results))
	assert.New(t).Equal(MaxDeviationsPerPage, len(tagResultsConsoleExploit.Results))
}
