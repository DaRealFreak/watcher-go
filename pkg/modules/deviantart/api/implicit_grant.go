package api

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/DaRealFreak/watcher-go/pkg/models"
	implicitoauth2 "github.com/DaRealFreak/watcher-go/pkg/oauth2"
	"github.com/PuerkitoBio/goquery"
	"golang.org/x/oauth2"
)

// ImplicitGrantDeviantart is the implementation of the ImplicitGrant interface
type ImplicitGrantDeviantart struct {
	implicitoauth2.ImplicitGrant
	account *models.Account
}

// NewImplicitGrantDeviantart returns the ImplicitGrantDeviantArt struct implementing the Implicit Grant OAuth2 flow
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

// Login implements the interface function of the Implicit Grant OAuth2 flow for DeviantArt
func (g ImplicitGrantDeviantart) Login() error {
	res, err := g.Client.Get("https://www.deviantart.com/users/login")
	if err != nil {
		return err
	}

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

// Authorize implements the interface function of the Implicit Grant OAuth2 flow for DeviantArt (only new style)
func (g ImplicitGrantDeviantart) Authorize() error {
	res, err := g.Client.Get(implicitoauth2.AuthTokenURL(g.Config, "session-id"))
	if err != nil {
		return err
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return err
	}

	form := doc.Find("form[action*='authorize_app']").First()

	// scrape all relevant data from the selected form
	authValues := new(authInfo)
	authValues.State, _ = form.Find("input[name=\"state\"]").First().Attr("value")
	authValues.ResponseType, _ = form.Find("input[name=\"response_type\"]").First().Attr("value")
	authValues.CSRFToken, _ = form.Find("input[name=\"csrf_token\"]").First().Attr("value")

	// pack it into values and send the post request
	values := url.Values{
		"referer":       {authValues.State},
		"csrf_token":    {authValues.CSRFToken},
		"client_id":     {g.Config.ClientID},
		"response_type": {authValues.ResponseType},
		"redirect_uri":  {g.Config.RedirectURL},
		"scope":         {strings.Join(g.Config.Scopes, " ")},
		"state":         {authValues.State},
		"authorized":    {"1"},
	}

	// the custom redirect function is still active, so new check is executed here
	_, err = g.Client.PostForm("https://www.deviantart.com/_sisu/do/authorize_app", values)
	if err != nil {
		return err
	}

	return nil
}
