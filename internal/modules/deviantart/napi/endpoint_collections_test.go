package napi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeviantartNAPI_CollectionsUser(t *testing.T) {
	res, err := daNAPI.CollectionsUser("GeneralDelta", 0, 50, FolderTypeFavourites, false, false)
	assert.New(t).NoError(err)
	assert.New(t).Equal(50, len(res.Collections))
	assert.New(t).Equal(true, res.HasMore)

	res, err = daNAPI.CollectionsUser("GeneralDelta", 0, 50, FolderTypeGallery, false, false)
	assert.New(t).NoError(err)
	assert.New(t).NotEqual(50, len(res.Collections))
	assert.New(t).Equal(false, res.HasMore)
}

func TestDeviantartNAPI_CollectionSearch(t *testing.T) {
	res, err := daNAPI.CollectionSearch("Aunt Cass", "", "")
	assert.New(t).NoError(err)
	assert.New(t).Equal(24, len(res.Collections))
	assert.New(t).Equal(true, res.HasMore)
}
