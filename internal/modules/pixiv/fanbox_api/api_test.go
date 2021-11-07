package fanboxapi

import (
	"os"
	"testing"

	"github.com/DaRealFreak/watcher-go/internal/models"
)

// getTestFanboxAPI builds the Fanbox API from the environment variables and executes all required functions
func getTestFanboxAPI() *FanboxAPI {
	fanboxAPI := NewFanboxAPI("pixiv Fanbox API")
	fanboxAPI.SessionCookie = &models.Cookie{Value: os.Getenv("PIXIV_SESSION_ID")}

	fanboxAPI.AddRoundTrippers()

	return fanboxAPI
}

// TestLogin tests
func TestLogin(t *testing.T) {
	// ToDo: check if sessionCookie is correct
}
