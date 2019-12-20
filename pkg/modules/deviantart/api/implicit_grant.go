package api

import (
	"fmt"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	implicitoauth2 "github.com/DaRealFreak/watcher-go/pkg/oauth2"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
	"golang.org/x/oauth2"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

type ImplicitGrantDeviantart struct {
	implicitoauth2.ImplicitGrant
	account *models.Account
}

func NewImplicitGrantDeviantart(
	cfg *oauth2.Config, client *http.Client, account *models.Account,
) *ImplicitGrantDeviantart {
	implicitGrantDeviantart := ImplicitGrantDeviantart{
		ImplicitGrant: implicitoauth2.ImplicitGrant{
			Config: cfg,
			Client: client,
		},
		account: account,
	}
	// register our current struct as the interface for the ImplicitGrant
	implicitGrantDeviantart.ImplicitGrant.ImplicitGrantInterface = implicitGrantDeviantart

	return &implicitGrantDeviantart
}

func (g ImplicitGrantDeviantart) Login() error {
	res, err := g.Client.Get("https://www.deviantart.com/users/login")
	raven.CheckError(err)

	info, err := g.getLoginCSRFToken(res)
	if err != nil {
		return err
	}

	if !(info.CSRFToken != "") {
		return fmt.Errorf("could not retrieve CSRF token from login page")
	}

	values := url.Values{
		"referer":    {"https://www.deviantart.com"},
		"csrf_token": {info.CSRFToken},
		"challenge":  {"0"},
		"username":   {g.account.Username},
		"password":   {g.account.Password},
		"remember":   {"on"},
	}

	res, err = g.Client.PostForm("https://www.deviantart.com/_sisu/do/signin", values)
	if err != nil {
		return err
	}

	content, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if !strings.Contains(string(content), "\"loggedIn\":true") &&
		!strings.Contains(string(content), "\\\"isLoggedIn\\\":true") {
		return fmt.Errorf("login failed")
	}

	return nil
}

func (g ImplicitGrantDeviantart) Authorize() {
	res, err := g.Client.Get(implicitoauth2.AuthCodeURLImplicit(g.Config, "session-id"))

	switch errorWrap := err.(type) {
	case *url.Error:
		switch errorWrap.Err.(type) {
		case implicitoauth2.ImplicitGrantRedirect:
			// already redirected to the token URL, no need for authorization
			return
		}
	}

	fmt.Println("todo: authorize")
	fmt.Println(res)
}
