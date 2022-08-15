package napi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeviantartNAPI_CollectionSearch(t *testing.T) {
	res, err := daNAPI.CollectionSearch("Aunt Cass", "", "")
	assert.New(t).NoError(err)
	assert.New(t).Equal(24, len(res.Collections))
	assert.New(t).Equal(true, res.HasMore)
}
