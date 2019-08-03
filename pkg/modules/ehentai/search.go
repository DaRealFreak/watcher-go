package ehentai

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/kubernetes/klog"
	"strings"
	"watcher-go/pkg/models"
)

type searchGalleryItem struct {
	id  string
	uri string
}

func (m *ehentai) parseSearch(item *models.TrackedItem) {
	response, _ := m.Session.Get(item.Uri, 0)
	html, _ := m.Session.GetDocument(response).Html()

	var itemQueue []searchGalleryItem
	foundCurrentItem := false
	for foundCurrentItem == false {
		for _, galleryItem := range m.getSearchGalleryUrls(html) {
			if galleryItem.id != item.CurrentItem {
				itemQueue = append(itemQueue, galleryItem)
			} else {
				foundCurrentItem = true
				break
			}
		}

		// break outer loop too if the current item got found
		if foundCurrentItem {
			break
		}

		nextPageUrl, exists := m.getNextSearchPageUrl(html)
		if exists == false {
			// no next page exists anymore, break here
			break
		}
		response, _ = m.Session.Get(nextPageUrl, 0)
		html, _ = m.Session.GetDocument(response).Html()
	}

	// reverse to add oldest items first
	for i, j := 0, len(itemQueue)-1; i < j; i, j = i+1, j-1 {
		itemQueue[i], itemQueue[j] = itemQueue[j], itemQueue[i]
	}
	// add items
	for _, gallery := range itemQueue {
		klog.Info("added gallery to tracked items: " + gallery.uri)
		m.DbIO.GetFirstOrCreateTrackedItem(gallery.uri, m)
	}

}

func (m *ehentai) getSearchGalleryUrls(html string) []searchGalleryItem {
	var items []searchGalleryItem

	document, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	document.Find("table.itg td.gl3c a[href]").Each(func(index int, row *goquery.Selection) {
		uri, _ := row.Attr("href")
		items = append(items, searchGalleryItem{
			id:  m.searchGalleryIdPattern.FindString(uri),
			uri: uri,
		})
	})

	return items
}

func (m *ehentai) getNextSearchPageUrl(html string) (url string, exists bool) {
	document, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	pages := document.Find("table.ptb td")
	pages = pages.Slice(pages.Length()-1, pages.Length())
	return pages.Find("a[href]").Attr("href")
}
