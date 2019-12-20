package implicitoauth2

import (
	"errors"
	"fmt"
	"golang.org/x/oauth2"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
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

type ImplicitGrantRedirect struct {
	URL *url.URL
}

func (r ImplicitGrantRedirect) Error() string {
	return fmt.Sprintf("reached Token URL: %s", r.URL)
}

func (g ImplicitGrant) Token() (token *oauth2.Token, err error) {
	if err := g.Login(); err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	var fragments url.Values

	wg.Add(1)

	go func() {
		<-time.After(time.Second * 2)
		err = fmt.Errorf("implicit grant request timeout")
		wg.Done()
	}()

	g.Client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		fragments, _ = url.ParseQuery(req.URL.Fragment)
		if fragments.Get("access_token") != "" {
			wg.Done()

			return ImplicitGrantRedirect{URL: req.URL}
		}

		// default redirect taken from http.Client (client.go) defaultRedirect
		if len(via) >= 10 {
			return errors.New("stopped after 10 redirects")
		}
		return nil
	}

	if err := g.Authorize(); err != nil && !ImplicitGrantRequestErrorSuccessful(err) {
		wg.Done()
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

func ImplicitGrantRequestErrorSuccessful(err error) bool {
	switch errorWrap := err.(type) {
	case *url.Error:
		switch errorWrap.Err.(type) {
		case ImplicitGrantRedirect:
			// already redirected to the token URL, no need for authorization
			return true
		}
	}

	return false
}
