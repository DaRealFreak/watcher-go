package napi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeviationSearch(t *testing.T) {
	res, err := daNAPI.DeviationSearch("Aunt Cass", "", "")
	assert.New(t).NoError(err)
	assert.New(t).Equal(24, len(res.Deviations))
	assert.New(t).Equal(true, res.HasMore)

	println(len(res.Deviations))
}
