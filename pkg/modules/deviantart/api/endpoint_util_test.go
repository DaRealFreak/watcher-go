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

	// toggle console exploit, we also require the first OAuth2 process to have succeeded
	// since we require the user information cookie which is set on a successful login
	daAPI.useConsoleExploit = true

	placeboConsoleExploit, err := daAPI.Placebo()
	assert.New(t).NoError(err)
	assert.New(t).Equal("success", placebo.Status)

	assert.New(t).Equal(placebo, placeboConsoleExploit)
}
