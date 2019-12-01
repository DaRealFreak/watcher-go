package jinjamodoki

import (
	"net/url"
	"strconv"

	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/PuerkitoBio/goquery"
)

// parsePage parses a user page for contributions
func (m *jinjaModoki) parsePage(item *models.TrackedItem) error {
	foundCurrent := false
	currentPageURI := item.URI

	for !foundCurrent {
		res, err := m.Session.Get(currentPageURI)
		if err != nil {
			return err
		}

		doc := m.Session.GetDocument(res)
		nextPage := m.findNextPage(doc)
		currentPageOffset := m.getNavigationOffset(currentPageURI)

		if nextPage.Length() > 0 {
			link, _ := nextPage.First().Attr("href")

			nextPageRelativeURI, err := url.Parse(link)
			if err != nil {
				return err
			}

			currentPageURI = m.baseURL.ResolveReference(nextPageRelativeURI).String()
			nextPageOffset := m.getNavigationOffset(currentPageURI)

			// the latest navigation link is the previous link, so we are done here
			if nextPageOffset <= currentPageOffset {
				break
			}
		}
	}

	return nil
}

// findNextPage returns the HTML selection to the next page if existing
func (m *jinjaModoki) findNextPage(doc *goquery.Document) *goquery.Selection {
	return doc.Find("div.list_all > form a[href*='offset']:last-child")
}

// getNavigationOffset returns the navigation offset from the passed URI GET parameters (0 if not set)
func (m *jinjaModoki) getNavigationOffset(navigationURI string) int {
	u, _ := url.Parse(navigationURI)

	q, _ := url.ParseQuery(u.RawQuery)
	if len(q["doffset"]) == 0 {
		return 0
	}

	// on error it would return 0 too, which is our default if not set anyways
	offset, _ := strconv.ParseInt(q["doffset"][0], 10, 64)

	return int(offset)
}
