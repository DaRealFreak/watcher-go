package linkfinder

import (
	"regexp"
)

func getLinksFromText(text string) (links []string) {
	pattern := regexp.MustCompile(`(?m)https?://[-a-zA-Z0-9@:%._+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_+.~#?&/=!]*)`)

	urlMatches := pattern.FindAllStringSubmatch(text, -1)
	for _, match := range urlMatches {
		links = append(links, match[0])
	}

	return links
}
