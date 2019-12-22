package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeviantartAPI_BrowseTags(t *testing.T) {
	daAPI := getTestDeviantartAPI()

	tagResults, err := daAPI.BrowseTags("test", 0, MaxDeviationsPerPage)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(tagResults)

	// toggle console exploit, we also require the first OAuth2 process to have succeeded
	// since we require the user information cookie which is set on a successful login
	daAPI.useConsoleExploit = true

	tagResultsConsoleExploit, err := daAPI.BrowseTags("test", 0, MaxDeviationsPerPage)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(tagResultsConsoleExploit)

	// console API results are NOT cached and can contain already deleted items
	// so comparison of API result and console exploit API results are differentiating
	// so we just ensure that the amount of results in page 1 is 24
	assert.New(t).Equal(MaxDeviationsPerPage, len(tagResults.Results))
	assert.New(t).Equal(MaxDeviationsPerPage, len(tagResultsConsoleExploit.Results))
}
