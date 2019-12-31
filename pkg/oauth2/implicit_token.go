package implicitoauth2

import (
	"sync"

	"golang.org/x/oauth2"
)

// ImplicitGrantTokenSource is the interface implementation of the oauth2.TokenSource interface
type ImplicitGrantTokenSource struct {
	Grant ImplicitGrantInterface
	mu    sync.Mutex // guards t
	token *oauth2.Token
}

// Token is the implementation of the TokenSource interface to return a valid token or the error occurred
// most of this functionality is copied from the oauth2 package from google
func (s *ImplicitGrantTokenSource) Token() (_ *oauth2.Token, err error) {
	s.mu.Lock()

	defer s.mu.Unlock()

	if s.token.Valid() {
		return s.token, nil
	}

	s.token, err = s.Grant.Token()
	if err != nil {
		return nil, err
	}

	return s.token, nil
}
