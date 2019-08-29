package session

import "golang.org/x/oauth2"

// TokenStore manages the tokens for the scopes
type TokenStore struct {
	tokens map[string]*oauth2.Token
}

// NewTokenStore returns initializes the token map and returns the store
func NewTokenStore() *TokenStore {
	return &TokenStore{
		tokens: map[string]*oauth2.Token{},
	}
}

// HasToken checks if a token for the specified scope is already set
func (t *TokenStore) HasToken(scope string) bool {
	_, exists := t.tokens[scope]
	return exists
}

// GetToken returns the OAuth2 token for the passed scope
func (t *TokenStore) GetToken(scope string) *oauth2.Token {
	val, exists := t.tokens[scope]
	if exists {
		return val
	}
	return nil
}

// SetToken sets the OAuth2 token for the passed scope
func (t *TokenStore) SetToken(scope string, token *oauth2.Token) {
	t.tokens[scope] = token
}
