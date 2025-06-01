package login

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	http "github.com/bogdanfinn/fhttp"
	"regexp"
)

type DeviantArtLogin struct {
}

// Info contains all relevant information from the login page
type Info struct {
	CSRFToken string `json:"csrfToken"`
	LuToken   string `json:"luToken"`
	LuToken2  string `json:"luToken2"`
}

// GetLoginCSRFToken returns the CSRF token from the login site to use in our POST login request
func (g DeviantArtLogin) GetLoginCSRFToken(res *http.Response) (*Info, error) {
	document, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, err
	}

	luToken, _ := document.Find(
		"form[action='/_sisu/do/step2'] input[name='lu_token'], " +
			"form[action='/_sisu/do/signin'] input[name='lu_token']",
	).First().Attr("value")

	// luToken2 is not only present and required for the signin form
	luToken2, _ := document.Find(
		"form[action='/_sisu/do/step2'] input[name='lu_token2'], " +
			"form[action='/_sisu/do/signin'] input[name='lu_token2']",
	).First().Attr("value")

	csrfToken, exists := document.Find(
		"form[action='/_sisu/do/step2'] input[name='csrf_token'], " +
			"form[action='/_sisu/do/signin'] input[name='csrf_token']",
	).First().Attr("value")

	if !exists {
		// If the csrf_token is not found in the form, we try to extract it from the HTML
		// (available everywhere, not just on the login page)
		html, _ := document.Html()
		re := regexp.MustCompile(`window\.__CSRF_TOKEN__\s*=\s*'([^']+)';`)
		matches := re.FindStringSubmatch(html)
		if len(matches) < 2 {
			return nil, fmt.Errorf("CSRF token not found")
		}
		csrfToken = matches[1]
	}

	return &Info{
		CSRFToken: csrfToken,
		LuToken:   luToken,
		LuToken2:  luToken2,
	}, nil
}
