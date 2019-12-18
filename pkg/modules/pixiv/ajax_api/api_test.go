package ajaxapi

import (
	"os"
	"testing"

	"github.com/DaRealFreak/watcher-go/pkg/models"
)

// getTestAjaxAPI builds the AJAX API from the environment variables and executes all required functions
func getTestAjaxAPI() *AjaxAPI {
	ajaxAPI := NewAjaxAPI("pixiv AJAX API")
	ajaxAPI.SessionCookie = &models.Cookie{Value: os.Getenv("PIXIV_SESSION_ID")}

	ajaxAPI.AddRoundTrippers()

	return ajaxAPI
}

// TestLogin tests
func TestLogin(t *testing.T) {
	// ToDo: check if sessionCookie are correct
}
