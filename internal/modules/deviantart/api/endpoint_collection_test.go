package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeviantartAPI_Collection(t *testing.T) {
	daAPI.useConsoleExploit = false

	collection, err := daAPI.Collection(
		"CLG-Artisa", "338AC44C-9373-061A-364C-DAC39C26935C", 0, MaxDeviationsPerPage,
	)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(collection)

	// toggle console exploit, we also require the first OAuth2 process to have succeeded
	// since we require the user information cookie which is set on a successful login
	daAPI.useConsoleExploit = true

	tagResultsConsoleExploit, err := daAPI.Collection(
		"CLG-Artisa", "338AC44C-9373-061A-364C-DAC39C26935C", 0, MaxDeviationsPerPage,
	)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(tagResultsConsoleExploit)

	// console API results are NOT cached and can contain already deleted items
	// so comparison of API result and console exploit API results are differentiating
	// so we just ensure that the amount of results in page 1 is 24
	assert.New(t).Equal(MaxDeviationsPerPage, len(collection.Results))
	assert.New(t).Equal(MaxDeviationsPerPage, len(tagResultsConsoleExploit.Results))
}

func TestDeviantartAPI_CollectionFolders(t *testing.T) {
	daAPI.useConsoleExploit = false

	folders, err := daAPI.CollectionFolders("CLG-Artisa", 0, MaxDeviationsPerPage)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(folders)

	// toggle console exploit, we also require the first OAuth2 process to have succeeded
	// since we require the user information cookie which is set on a successful login
	daAPI.useConsoleExploit = true

	foldersConsoleExploit, err := daAPI.CollectionFolders("CLG-Artisa", 0, MaxDeviationsPerPage)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(foldersConsoleExploit)

	assert.New(t).Equal(len(folders.Results), len(foldersConsoleExploit.Results))
}

func TestDeviantartAPI_CollectionNameFromID(t *testing.T) {
	daAPI.useConsoleExploit = false

	collectionTitle, err := daAPI.CollectionNameFromID("clg-artisa", 80472763)
	assert.New(t).NoError(err)
	assert.New(t).Equal("Deep Blue II", collectionTitle)

	// toggle console exploit, we also require the first OAuth2 process to have succeeded
	// since we require the user information cookie which is set on a successful login
	daAPI.useConsoleExploit = true

	collectionTitleConsoleExploit, err := daAPI.CollectionNameFromID("clg-artisa", 80472763)
	assert.New(t).NoError(err)
	assert.New(t).Equal("Deep Blue II", collectionTitleConsoleExploit)
}

func TestDeviantartAPI_CollectionFolderIDToUUID(t *testing.T) {
	daAPI.useConsoleExploit = false

	folderUUID, err := daAPI.CollectionFolderIDToUUID("clg-artisa", 80472763)
	assert.New(t).NoError(err)
	assert.New(t).Equal("338AC44C-9373-061A-364C-DAC39C26935C", folderUUID)

	// toggle console exploit, we also require the first OAuth2 process to have succeeded
	// since we require the user information cookie which is set on a successful login
	daAPI.useConsoleExploit = true

	folderUUIDConsoleExploit, err := daAPI.CollectionFolderIDToUUID("clg-artisa", 80472763)
	assert.New(t).NoError(err)
	assert.New(t).Equal("338AC44C-9373-061A-364C-DAC39C26935C", folderUUIDConsoleExploit)
}
