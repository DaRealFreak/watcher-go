package api

import (
	"context"
	watcherHttp "github.com/DaRealFreak/watcher-go/pkg/http"
	"github.com/DaRealFreak/watcher-go/pkg/http/session"
	"golang.org/x/oauth2"
	"golang.org/x/time/rate"
	"time"
)

type DeviantartAPI struct {
	Session      watcherHttp.SessionInterface
	rateLimiter  *rate.Limiter
	ctx          context.Context
	OAuth2Config *oauth2.Config
}

func NewDeviantartAPI(moduleKey string) *DeviantartAPI {
	return &DeviantartAPI{
		Session: session.NewSession(moduleKey),
		OAuth2Config: &oauth2.Config{
			ClientID: "9991",
			Endpoint: oauth2.Endpoint{
				AuthURL: "https://www.deviantart.com/oauth2/authorize",
			},
			Scopes:      []string{"basic", "browse", "gallery", "feed"},
			RedirectURL: "https://lvh.me/da-cb",
		},
		rateLimiter: rate.NewLimiter(rate.Every(1500*time.Millisecond), 1),
		ctx:         context.Background(),
	}
}
