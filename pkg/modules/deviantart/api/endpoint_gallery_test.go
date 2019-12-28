package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeviantartAPI_Gallery(t *testing.T) {
	daAPI.useConsoleExploit = false

	gallery, err := daAPI.Gallery(
		"CLG-Artisa", 80472763, 0, MaxDeviationsPerPage,
	)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(gallery)

	// toggle console exploit, we also require the first OAuth2 process to have succeeded
	// since we require the user information cookie which is set on a successful login
	daAPI.useConsoleExploit = true

	galleryConsoleExploit, err := daAPI.Gallery(
		"CLG-Artisa", 80472763, 0, MaxDeviationsPerPage,
	)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(galleryConsoleExploit)

	assert.New(t).Equal(MaxDeviationsPerPage, len(gallery.Results))
	assert.New(t).Equal(MaxDeviationsPerPage, len(galleryConsoleExploit.Results))
}

func TestDeviantartAPI_GalleryAll(t *testing.T) {
	daAPI.useConsoleExploit = false

	gallery, err := daAPI.GalleryAll(
		"CLG-Artisa", 0, MaxDeviationsPerPage,
	)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(gallery)

	// toggle console exploit, we also require the first OAuth2 process to have succeeded
	// since we require the user information cookie which is set on a successful login
	daAPI.useConsoleExploit = true

	galleryConsoleExploit, err := daAPI.GalleryAll(
		"CLG-Artisa", 0, MaxDeviationsPerPage,
	)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(galleryConsoleExploit)

	assert.New(t).Equal(MaxDeviationsPerPage, len(gallery.Results))
	assert.New(t).Equal(MaxDeviationsPerPage, len(galleryConsoleExploit.Results))
}
