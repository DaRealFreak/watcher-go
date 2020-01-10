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

func TestDeviantartAPI_DeviationDownload(t *testing.T) {
	daAPI.useConsoleExploit = false

	deviationDownload, err := daAPI.DeviationDownload("E75CDA3B-3160-3CC6-4774-7CD503260472")
	assert.New(t).NoError(err)
	assert.New(t).NotNil(deviationDownload)

	// toggle console exploit, we also require the first OAuth2 process to have succeeded
	// since we require the user information cookie which is set on a successful login
	daAPI.useConsoleExploit = true

	deviationDownloadConsoleExploit, err := daAPI.DeviationDownload("E75CDA3B-3160-3CC6-4774-7CD503260472")
	assert.New(t).NoError(err)
	assert.New(t).NotNil(deviationDownloadConsoleExploit)
}

func TestDeviantartAPI_DeviationDownloadFallback(t *testing.T) {
	// special case for web frontend requests:
	// the cookie will only get set on the first OAuth2 authentication since it'll get set during the round trip
	// where the used cookies for the request are already set, so the request would not display a download link
	// so we call the placebo function for the OAuth2 process, this special case does not occur in the real application
	_, err := daAPI.Placebo()
	assert.New(t).NoError(err)

	deviationDownload, err := daAPI.DeviationDownloadFallback(
		"https://www.deviantart.com/yakovlev-vad/art/Landscape-3-500009916",
	)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(deviationDownload)
}
