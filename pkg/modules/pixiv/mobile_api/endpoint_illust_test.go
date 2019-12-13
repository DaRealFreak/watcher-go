package mobileapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewMobileAPI(t *testing.T) {
	mobileAPI := getTestMobileAPI()

	illustDetail, err := mobileAPI.GetIllustDetail(123456)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(illustDetail)
}
