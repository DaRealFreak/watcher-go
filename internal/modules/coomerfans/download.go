package coomerfans

import "github.com/PuerkitoBio/goquery"

// extractPostMedia returns the content media URLs of a post page: images and
// video sources inside `div.post-body`. Avatar/recommended thumbnails live
// outside that container and are excluded. URLs are returned in document
// order (images before video sources), de-duplicated.
func extractPostMedia(doc *goquery.Document) []string {
	var urls []string
	seen := make(map[string]bool)
	add := func(u string) {
		if u != "" && !seen[u] {
			seen[u] = true
			urls = append(urls, u)
		}
	}
	doc.Find("div.post-body img[src]").Each(func(_ int, s *goquery.Selection) {
		src, _ := s.Attr("src")
		add(src)
	})
	doc.Find("div.post-body video source[src]").Each(func(_ int, s *goquery.Selection) {
		src, _ := s.Attr("src")
		add(src)
	})
	return urls
}

// scrapeUsername returns the creator username from the first `/u/` link on a
// post page, or "" if none is present. Used to derive the download folder for
// single-post tracked items (which carry no creator subfolder).
func scrapeUsername(doc *goquery.Document) string {
	var username string
	doc.Find("a[href^='/u/']").EachWithBreak(func(_ int, s *goquery.Selection) bool {
		href, ok := s.Attr("href")
		if !ok {
			return true
		}
		if _, _, name, ok := parseUserURL(href); ok {
			username = name
			return false
		}
		return true
	})
	return username
}
