package napi

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/DaRealFreak/watcher-go/internal/models"
)

// nolint: gochecknoglobals
var daNAPI *DeviantartNAPI

// TestMain is the constructor for the test functions to use a shared API instance
// to prevent multiple logins for every test
func TestMain(m *testing.M) {
	testAccount := &models.Account{
		Username: os.Getenv("DEVIANTART_USER"),
		Password: os.Getenv("DEVIANTART_PASS"),
	}

	// initialize the shared API instance
	daNAPI = NewDeviantartNAPI("deviantart API", testAccount)
	daNAPI.AddRoundTrippers("")

	// run the unit tests
	os.Exit(m.Run())
}

func TestNewDeviantartAPI(t *testing.T) {
	res, err := daNAPI.DeviationSearch("Aunt Cass", "", "")
	assert.New(t).NoError(err)
	assert.New(t).Equal(24, len(res.Deviations))
	assert.New(t).Equal(true, res.HasMore)

	println(len(res.Deviations))
}
