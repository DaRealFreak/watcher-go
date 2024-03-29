package login

import (
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type DeviantArtLogin struct {
}

// Info contains all relevant information from the login page
type Info struct {
	CSRFToken string `json:"csrfToken"`
}

// GetLoginCSRFToken returns the CSRF token from the login site to use in our POST login request
func (g DeviantArtLogin) GetLoginCSRFToken(res *http.Response) (*Info, error) {
	var currentLoginInfo Info

	jsonPattern := regexp.MustCompile(`.*\\"csrfToken\\":\\"(?P<Number>[^\\"]+)\\".*`)

	document, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, err
	}

	scriptTags := document.Find("script")
	scriptTags.Each(func(row int, selection *goquery.Selection) {
		// no need for further checks if we already have our login info
		if currentLoginInfo.CSRFToken != "" {
			return
		}

		scriptContent := selection.Text()
		if jsonPattern.MatchString(scriptContent) {
			currentLoginInfo.CSRFToken = jsonPattern.FindStringSubmatch(scriptContent)[1]
		}
	})

	html, parseErr := document.Html()
	if parseErr != nil {
		return nil, parseErr
	}

	// set body again in case we have to read from the body again
	res.Body = io.NopCloser(strings.NewReader(html))

	return &currentLoginInfo, err
}
