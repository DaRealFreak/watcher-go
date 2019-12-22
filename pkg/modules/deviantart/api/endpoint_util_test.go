package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeviantartAPI_Placebo(t *testing.T) {
	daAPI := getTestDeviantartAPI()

	placebo, err := daAPI.Placebo()
	assert.New(t).NoError(err)
	assert.New(t).Equal("success", placebo.Status)
}
