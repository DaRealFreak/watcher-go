// Package implicitoauth2 contains the basic functionality to realize an Implicit Grant OAuth2 authentication
package implicitoauth2

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

// ImplicitGrantInterface defines all required methods for an implicit grant OAuth2 process
type ImplicitGrantInterface interface {
	Login() error
	Authorize() error
	Token() (*oauth2.Token, error)
}

// ImplicitGrant implements the ImplicitGrantInterface and provides a basic token function using the interface methods
type ImplicitGrant struct {
	ImplicitGrantInterface
	Client *http.Client
	Config *oauth2.Config
}

// ImplicitGrantRedirect is a custom error we return when we got successfully redirected to a Token URL
type ImplicitGrantRedirect struct {
	URL *url.URL
}

// Error is the implementation of the default output of the error interface
func (r ImplicitGrantRedirect) Error() string {
	return fmt.Sprintf("reached Token URL: %s", r.URL)
}

// Token is a basic implementation for the workflow of the Implicit Grant OAuth2 authentication
func (g ImplicitGrant) Token() (token *oauth2.Token, err error) {
	if err := g.Login(); err != nil {
		return nil, err
	}

	var (
		wg        sync.WaitGroup
		fragments url.Values
	)

	done := make(chan struct{})

	wg.Add(1)

	go func() {
		select {
		case <-done:
			return
		case <-time.After(time.Second * 5):
			err = fmt.Errorf("implicit grant request timeout")
			wg.Done()
		}
	}()

	g.Client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		fragments, _ = url.ParseQuery(req.URL.Fragment)
		if fragments.Get("access_token") != "" {
			wg.Done()
			// cancel our go routine too instead of letting it run into the timeout
			defer close(done)

			return ImplicitGrantRedirect{URL: req.URL}
		}

		// default redirect taken from http.Client (client.go) defaultRedirect
		if len(via) >= 10 {
			return errors.New("stopped after 10 redirects")
		}

		return nil
	}

	if err := g.Authorize(); err != nil && !ImplicitGrantRequestErrorSuccessful(err) {
		return nil, err
	}

	wg.Wait()

	expires, err := strconv.Atoi(fragments.Get("expires_in"))

	return &oauth2.Token{
		AccessToken:  fragments.Get("access_token"),
		TokenType:    fragments.Get("token_type"),
		RefreshToken: fragments.Get("refresh_token"),
		Expiry:       time.Now().Add(time.Duration(expires) * time.Second),
	}, err
}

// ImplicitGrantRequestErrorSuccessful checks if the error occurred is the error indicating a successful redirect
// which is returned as error since we cancel the request the moment we reach it
func ImplicitGrantRequestErrorSuccessful(err error) bool {
	if x, ok := err.(*url.Error); ok {
		if _, isRedirect := x.Err.(ImplicitGrantRedirect); isRedirect {
			// already redirected to the token URL, no need for authorization
			return true
		}
	}

	return false
}
