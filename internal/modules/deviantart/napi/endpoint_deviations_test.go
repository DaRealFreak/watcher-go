package napi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeviantartNAPI_DeviationSearch(t *testing.T) {
	res, err := daNAPI.DeviationSearch("Aunt Cass", "", "")
	assert.New(t).NoError(err)
	assert.New(t).Equal(24, len(res.Deviations))
	assert.New(t).Equal(true, res.HasMore)
}

func TestDeviantartNAPI_DeviationTag(t *testing.T) {
	res, err := daNAPI.DeviationTag("pyra", "", "")
	assert.New(t).NoError(err)
	assert.New(t).Equal(24, len(res.Deviations))
	assert.New(t).Equal(true, res.HasMore)
}
