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

func TestMobileAPI_GetUserIllusts(t *testing.T) {
	mobileAPI := getTestMobileAPI()

	userIllusts, err := mobileAPI.GetUserIllusts(7210261, "", 0)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(userIllusts)
	assert.New(t).Equal(len(userIllusts.Illustrations), 30)
	assert.New(t).NotEmpty(userIllusts.NextURL)
}

func TestMobileAPI_GetUserIllustsByURL(t *testing.T) {
	mobileAPI := getTestMobileAPI()

	userIllusts, err := mobileAPI.GetUserIllusts(7210261, "", 0)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(userIllusts)
	assert.New(t).Equal(len(userIllusts.Illustrations), 30)
	assert.New(t).NotEmpty(userIllusts.NextURL)

	nextPageUserIllusts, err := mobileAPI.GetUserIllustsByURL(userIllusts.NextURL)
	assert.New(t).NoError(err)
	assert.New(t).NotNil(nextPageUserIllusts)
	assert.New(t).Equal(len(nextPageUserIllusts.Illustrations), 30)
}
