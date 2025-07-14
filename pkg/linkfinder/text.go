package linkfinder

import (
	"net/url"
	"regexp"
	"strings"
)

func getLinksFromText(text string) (links []string) {
	pattern := regexp.MustCompile(`(?m)https?://[-a-zA-Z0-9@:%._+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_+.~#?&/=!]*)`)

	urlMatches := pattern.FindAllStringSubmatch(text, -1)
	if len(urlMatches) > 0 {
		for _, match := range urlMatches {
			q, _ := url.Parse(match[0])
			q.Scheme = "https"

			if !strings.Contains(q.String(), ".fanbox.cc/") {
				links = append(links, q.String())
			}
		}
	}

	return links
}
