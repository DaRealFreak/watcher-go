package mobileapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMobileAPI_GetUgoiraMetadata(t *testing.T) {
	ugoiraMetadata, err := mobileAPI.GetUgoiraMetadata(100791744)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(ugoiraMetadata)
}
