package api

import (
	"fmt"
	"golang.org/x/oauth2"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

type ImplicitGrantInterface interface {
	Login() bool
	Authorize()
}

type ImplicitGrant struct {
	ImplicitGrantInterface
	Client *http.Client
}

func (g ImplicitGrant) Token() (token *oauth2.Token, err error) {
	if !g.Login() {
		return nil, fmt.Errorf("login failed")
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
		wg.Done()
		return fmt.Errorf("dont follow redirects")
	}

	g.Authorize()
	wg.Wait()

	expires, err := strconv.Atoi(fragments.Get("expires_in"))

	return &oauth2.Token{
		AccessToken:  fragments.Get("access_token"),
		TokenType:    fragments.Get("token_type"),
		RefreshToken: fragments.Get("refresh_token"),
		Expiry:       time.Now().Add(time.Duration(expires) * time.Second),
	}, err
}
