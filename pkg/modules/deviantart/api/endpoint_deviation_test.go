package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeviantartAPI_DeviationContent(t *testing.T) {
	daAPI.useConsoleExploit = false

	deviationContent, err := daAPI.DeviationContent("B4715B24-C571-55B2-68A4-5DAE1479DF28")
	assert.New(t).NoError(err)
	assert.New(t).NotNil(deviationContent)
	assert.New(t).NotEqual("", deviationContent.HTML)

	// toggle console exploit, we also require the first OAuth2 process to have succeeded
	// since we require the user information cookie which is set on a successful login
	daAPI.useConsoleExploit = true

	deviationContentConsoleExploit, err := daAPI.DeviationContent("B4715B24-C571-55B2-68A4-5DAE1479DF28")
	assert.New(t).NoError(err)
	assert.New(t).NotNil(deviationContentConsoleExploit)
	assert.New(t).NotEqual("", deviationContentConsoleExploit.HTML)
}
