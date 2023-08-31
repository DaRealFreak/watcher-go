package napi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFavoritesUser(t *testing.T) {
	res, err := daNAPI.FavoritesUser("Zerion", 0, 0, 25, true)
	assert.New(t).NoError(err)
	assert.New(t).Equal(25, len(res.Deviations))
	assert.New(t).Equal(true, res.HasMore)
}

func TestDeviationsUser(t *testing.T) {
	res, err := daNAPI.DeviationsUser("boreddude666", 0, 0, MaxLimit, true)
	assert.New(t).NoError(err)
	assert.New(t).Equal(MaxLimit, len(res.Deviations))
	assert.New(t).Equal(true, res.HasMore)
}

func TestDeviantartNAPI_WatchUser(t *testing.T) {
	res, err := daNAPI.WatchUser("Zerion", nil)
	assert.New(t).NoError(err)
	assert.New(t).Equal(true, res.Success)
}

func TestDeviantartNAPI_UnwatchUser(t *testing.T) {
	res, err := daNAPI.UnwatchUser("Zerion", nil)
	assert.New(t).NoError(err)
	assert.New(t).Equal(true, res.Success)
}
