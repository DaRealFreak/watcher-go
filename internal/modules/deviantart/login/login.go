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
	LuToken   string `json:"luToken"`
	LuToken2  string `json:"luToken2"`
}

// GetLoginCSRFToken returns the CSRF token from the login site to use in our POST login request
func (g DeviantArtLogin) GetLoginCSRFToken(res *http.Response) (*Info, error) {
	var currentLoginInfo Info

	jsonPattern := regexp.MustCompile(`.*\\"csrfToken\\":\\"(?P<Number>[^\\"]+)\\".*`)
	luTokenPattern := regexp.MustCompile(`.*\\"luToken\\":\\"(?P<Number>[^\\"]+)\\".*`)
	luToken2Pattern := regexp.MustCompile(`.*\\"luToken2\\":\\"(?P<Number>[^\\"]+)\\".*`)

	document, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, err
	}

	scriptTags := document.Find("script")
	scriptTags.Each(func(row int, selection *goquery.Selection) {
		// no need for further checks if we already have our login info
		if currentLoginInfo.CSRFToken != "" && currentLoginInfo.LuToken != "" {
			return
		}

		scriptContent := selection.Text()
		if jsonPattern.MatchString(scriptContent) {
			currentLoginInfo.CSRFToken = jsonPattern.FindStringSubmatch(scriptContent)[1]
		}

		if luTokenPattern.MatchString(scriptContent) {
			currentLoginInfo.LuToken = luTokenPattern.FindStringSubmatch(scriptContent)[1]
		}

		if luToken2Pattern.MatchString(scriptContent) {
			currentLoginInfo.LuToken2 = luToken2Pattern.FindStringSubmatch(scriptContent)[1]
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
