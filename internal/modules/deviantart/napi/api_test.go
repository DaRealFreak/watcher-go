package napi

import (
	"golang.org/x/time/rate"
	"os"
	"testing"
	"time"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/stretchr/testify/assert"
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
	daNAPI = NewDeviantartNAPI("deviantart API", rate.NewLimiter(rate.Every(time.Duration(4000)*time.Millisecond), 1))
	daNAPI.AddRoundTrippers("Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:103.0) Gecko/20100101 Firefox/103.0")
	err := daNAPI.Login(testAccount)
	if err != nil {
		daNAPI = nil
		println("unable to login")
		os.Exit(-1)
	}

	// run the unit tests
	os.Exit(m.Run())
}

func TestNewDeviantartAPI(t *testing.T) {
	assert.New(t).NotNil(daNAPI)
}
