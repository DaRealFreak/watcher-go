package linkfinder

import (
	"github.com/PuerkitoBio/goquery"
	html2 "golang.org/x/net/html"
	"strings"
)

func getLinksFromHtml(html string) (links []string) {
	document, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return links
	}

	document.Find("a[href]:not([href^='/'])").Each(func(index int, item *goquery.Selection) {
		href, exists := item.Attr("href")
		if exists {
			// find text nodes immediately after the <a> tag to extract possible anchors
			var afterAnchor string
			for sibling := item.Nodes[0].NextSibling; sibling != nil; sibling = sibling.NextSibling {
				if sibling.Type == html2.TextNode {
					afterAnchor = strings.TrimSpace(sibling.Data)
					// only take the immediate next text node
					break
				}
			}

			// combine href with the fragment
			fullURL := href
			if afterAnchor != "" && strings.HasPrefix(afterAnchor, "#") {
				fullURL += afterAnchor
			}

			if !strings.Contains(fullURL, ".fanbox.cc/") && !strings.Contains(fullURL, "discord.gg/") {
				links = append(links, fullURL)
			}
		}
	})

	return links
}
