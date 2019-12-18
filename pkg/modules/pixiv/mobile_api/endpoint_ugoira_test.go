package mobileapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMobileAPI_GetUgoiraMetadata(t *testing.T) {
	mobileAPI := getTestMobileAPI()

	ugoiraMetadata, err := mobileAPI.GetUgoiraMetadata(78315530)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(ugoiraMetadata)
}
