package mobileapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMobileAPI_GetUserDetail(t *testing.T) {
	mobileAPI := getTestMobileAPI()

	userDetail, err := mobileAPI.GetUserDetail(123456)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(userDetail)
}
