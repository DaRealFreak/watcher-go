package api

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeviantartAPI_Gallery(t *testing.T) {
	daAPI.useConsoleExploit = false

	fmt.Println(daAPI.GalleryFolderIDToUUID("CLG-Artisa", 66857455))

	gallery, err := daAPI.Gallery(
		"CLG-Artisa", 66857455, 0, MaxDeviationsPerPage,
	)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(gallery)

	// toggle console exploit, we also require the first OAuth2 process to have succeeded
	// since we require the user information cookie which is set on a successful login
	daAPI.useConsoleExploit = true

	galleryConsoleExploit, err := daAPI.Gallery(
		"CLG-Artisa", 66857455, 0, MaxDeviationsPerPage,
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

func TestDeviantartAPI_GalleryFolders(t *testing.T) {
	daAPI.useConsoleExploit = false

	folders, err := daAPI.GalleryFolders("CLG-Artisa", 0, MaxDeviationsPerPage)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(folders)

	// toggle console exploit, we also require the first OAuth2 process to have succeeded
	// since we require the user information cookie which is set on a successful login
	daAPI.useConsoleExploit = true

	foldersConsoleExploit, err := daAPI.GalleryFolders("CLG-Artisa", 0, MaxDeviationsPerPage)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(foldersConsoleExploit)

	assert.New(t).Equal(len(folders.Results), len(foldersConsoleExploit.Results))
}

func TestDeviantartAPI_GalleryFolderIDToUUID(t *testing.T) {
	daAPI.useConsoleExploit = false

	folderUUID, err := daAPI.GalleryFolderIDToUUID("CLG-Artisa", 66857455)
	assert.New(t).NoError(err)
	assert.New(t).Equal("66979124-8385-B572-E652-933338A076B2", folderUUID)

	// toggle console exploit, we also require the first OAuth2 process to have succeeded
	// since we require the user information cookie which is set on a successful login
	daAPI.useConsoleExploit = true

	folderUUIDConsoleExploit, err := daAPI.GalleryFolderIDToUUID("CLG-Artisa", 66857455)
	assert.New(t).NoError(err)
	assert.New(t).Equal("66979124-8385-B572-E652-933338A076B2", folderUUIDConsoleExploit)
}
