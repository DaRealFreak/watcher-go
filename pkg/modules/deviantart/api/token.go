package api

import (
	implicitoauth2 "github.com/DaRealFreak/watcher-go/pkg/oauth2"
	"golang.org/x/oauth2"
	"sync"
)

type deviantartTokenRefresher struct {
	new   implicitoauth2.ImplicitGrantInterface
	mu    sync.Mutex // guards t
	token *oauth2.Token
}

// Token is the implementation of the TokenSource interface to return a valid token or the error occurred
// most of this functionality is copied from the oauth2 package from google
func (s *deviantartTokenRefresher) Token() (_ *oauth2.Token, err error) {
	s.mu.Lock()

	defer s.mu.Unlock()

	if s.token.Valid() {
		return s.token, nil
	}

	s.token, err = s.new.Token()
	if err != nil {
		return nil, err
	}

	return s.token, nil
}
