package linkfinder

// GetLinks extracts all links from the given text, including those in HTML and plain text formats.
func GetLinks(text string) []string {
	links := getLinksFromHtml(text)
	for _, link := range getLinksFromText(text) {
		if !contains(links, link) {
			links = append(links, link)
		}
	}

	return links
}

func contains[T comparable](slice []T, v T) bool {
	for _, item := range slice {
		if item == v {
			return true
		}
	}
	return false
}
