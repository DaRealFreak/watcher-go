package fanboxapi

import (
	"os"
	"testing"

	"github.com/DaRealFreak/watcher-go/internal/models"
)

const (
	// EnvSessionID is the environment variable holding the FANBOXSESSID cookie value
	EnvSessionID = "PIXIV_SESSION_ID"
	// EnvCfClearance is the environment variable holding the cf_clearance cookie value
	EnvCfClearance = "PIXIV_CF_CLEARANCE"
	// EnvUserAgent is the environment variable holding the user agent the cf_clearance
	// cookie got issued for, it has to match or cloudflare rejects the cookie
	EnvUserAgent = "PIXIV_FANBOX_USER_AGENT"
)

// getTestFanboxAPI builds the Fanbox API from the environment variables and executes all required functions
func getTestFanboxAPI() *FanboxAPI {
	fanboxAPI := NewFanboxAPI("pixiv Fanbox API")
	fanboxAPI.SessionCookie = &models.Cookie{Value: os.Getenv(EnvSessionID)}
	fanboxAPI.UserAgent = os.Getenv(EnvUserAgent)

	if cfClearance := os.Getenv(EnvCfClearance); cfClearance != "" {
		fanboxAPI.CfClearanceCookie = &models.Cookie{Name: CookieCfClearance, Value: cfClearance}
	}

	fanboxAPI.AddRoundTrippers()

	return fanboxAPI
}

// skipWithoutCloudflareClearance skips the test if no cf_clearance cookie and matching user agent
// are configured. The post.info endpoint answers every request carrying a FANBOXSESSID cookie with
// a cloudflare challenge, the other endpoints are reachable with the session alone. Cloudflare
// validates the cf_clearance cookie against the IP and User-Agent it got issued for, so a cookie
// taken from a browser only works with that browser's user agent and from the same IP.
func skipWithoutCloudflareClearance(t *testing.T) {
	t.Helper()

	if os.Getenv(EnvCfClearance) == "" || os.Getenv(EnvUserAgent) == "" {
		t.Skipf("%s and %s are required, post.info answers authenticated requests with a cloudflare challenge",
			EnvCfClearance, EnvUserAgent)
	}
}

// TestLogin tests
func TestLogin(t *testing.T) {
	// ToDo: check if sessionCookie is correct
}
