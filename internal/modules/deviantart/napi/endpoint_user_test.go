package napi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFavoritesUser(t *testing.T) {
	res, err := daNAPI.DeviationsUser("Zerion", 0, 0, MaxLimit, true)
	assert.New(t).NoError(err)
	assert.New(t).Equal(MaxLimit, len(res.Deviations))
	assert.New(t).Equal(true, res.HasMore)
}

func TestDeviationsUser(t *testing.T) {
	res, err := daNAPI.DeviationsUser("boreddude666", 0, 0, MaxLimit, true)
	assert.New(t).NoError(err)
	assert.New(t).Equal(MaxLimit, len(res.Deviations))
	assert.New(t).Equal(true, res.HasMore)
}
