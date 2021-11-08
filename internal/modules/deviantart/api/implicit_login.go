package api

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"

	"github.com/PuerkitoBio/goquery"
)

// loginInfo contains all relevant information from the login page
type loginInfo struct {
	CSRFToken string `json:"csrfToken"`
}

// getLoginCSRFToken returns the CSRF token from the login site to use in our POST login request
func (g ImplicitGrantDeviantart) getLoginCSRFToken(res *http.Response) (*loginInfo, error) {
	var currentLoginInfo loginInfo

	jsonPattern := regexp.MustCompile(`JSON.parse\((?P<Number>.*csrfToken.*?)\);`)

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
			var s string

			s, err = strconv.Unquote(jsonPattern.FindStringSubmatch(scriptContent)[1])
			if err != nil {
				return
			}

			if err = json.Unmarshal([]byte(s), &currentLoginInfo); err != nil {
				return
			}
		}
	})

	return &currentLoginInfo, err
}
