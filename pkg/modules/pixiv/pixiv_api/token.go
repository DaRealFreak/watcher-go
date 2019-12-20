package pixivapi

import (
	"net/http"
	"net/url"
	"sync"

	"github.com/DaRealFreak/watcher-go/pkg/modules/pixiv/pixiv_api/internal"
	"golang.org/x/oauth2"
)

type pixivTokenRefresher struct {
	config *oauth2.Config
	mu     sync.Mutex // guards t
	token  *oauth2.Token
	client *http.Client
}

// Token is the implementation of the TokenSource interface to return a valid token or the error occurred
// most of this functionality is copied from the oauth2 package from google
func (s *pixivTokenRefresher) Token() (_ *oauth2.Token, err error) {
	s.mu.Lock()

	defer s.mu.Unlock()

	if s.token.Valid() {
		return s.token, nil
	}

	s.token, err = s.refreshToken()
	if err != nil {
		return nil, err
	}

	return s.token, nil
}

// refreshToken is the custom refresh functionality since we also require our custom post values here
func (s *pixivTokenRefresher) refreshToken() (*oauth2.Token, error) {
	v := url.Values{
		"device_token":   {"pixiv"},
		"get_secure_url": {"true"},
		"include_policy": {"true"},
		"client_id":      {s.config.ClientID},
		"client_secret":  {s.config.ClientSecret},
		// refresh token specific values
		"grant_type":    {"refresh_token"},
		"refresh_token": {s.token.RefreshToken},
	}

	res, err := s.client.PostForm(s.config.Endpoint.TokenURL, v)
	if err != nil {
		return nil, err
	}

	return internal.RetrieveTokenFromResponse(res)
}
