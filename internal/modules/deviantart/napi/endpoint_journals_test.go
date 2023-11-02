package napi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeviantartNAPI_JournalSearch(t *testing.T) {
	res, err := daNAPI.JournalSearch("Aunt Cass", "", "")
	assert.New(t).NoError(err)
	assert.New(t).Equal(24, len(res.Deviations))
	assert.New(t).Equal(true, res.HasMore)
}
