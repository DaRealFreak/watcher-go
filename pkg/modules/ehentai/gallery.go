package ehentai

import (
	"strings"

	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/PuerkitoBio/goquery"
)

// imageGalleryItem contains the relevant data of gallery items
type imageGalleryItem struct {
	id           string
	uri          string
	galleryTitle string
}

// parseGallery parses the tracked item if we detected a tracked gallery
func (m *ehentai) parseGallery(item *models.TrackedItem) {
	response, _ := m.Session.Get(item.URI)
	html, _ := m.Session.GetDocument(response).Html()
	if m.hasGalleryErrors(item, html) {
		m.DbIO.ChangeTrackedItemCompleteStatus(item, true)
		return
	}
	galleryTitle := m.extractGalleryTitle(html)

	var downloadQueue []imageGalleryItem
	foundCurrentItem := item.CurrentItem == ""

	for {
		for _, galleryItem := range m.getGalleryImageUrls(html, galleryTitle) {
			if foundCurrentItem {
				downloadQueue = append(downloadQueue, galleryItem)
			}
			// check if we reached the current item already
			if galleryItem.id == item.CurrentItem {
				foundCurrentItem = true
			}
		}

		nextPageURL, exists := m.getNextGalleryPageURL(html)
		if !exists {
			// no previous page exists anymore, break here
			break
		}
		response, _ = m.Session.Get(nextPageURL)
		html, _ = m.Session.GetDocument(response).Html()
	}

	m.processDownloadQueue(downloadQueue, item)
	if !m.downloadLimitReached {
		// mark item as complete since update doesn't affect old galleries
		// if download limit got reached we didn't reach the final image and don't set the gallery as complete
		m.DbIO.ChangeTrackedItemCompleteStatus(item, true)
	}
}

// getNextGalleryPageURL retrieves the url of the next page if it exists
func (m *ehentai) getNextGalleryPageURL(html string) (url string, exists bool) {
	document, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	pages := document.Find("table.ptb td")
	nextPageElement := pages.Slice(pages.Length()-1, pages.Length())
	return nextPageElement.Find("a[href]").Attr("href")
}

// getGalleryImageUrls extracts all gallery image urls from the passed html
func (m *ehentai) getGalleryImageUrls(html string, galleryTitle string) []imageGalleryItem {
	var imageUrls []imageGalleryItem
	document, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	document.Find("div#gdt > div a[href]").Each(func(index int, row *goquery.Selection) {
		uri, _ := row.Attr("href")
		imageUrls = append(imageUrls, imageGalleryItem{
			id:           m.galleryImageIDPattern.FindString(uri),
			uri:          uri,
			galleryTitle: galleryTitle,
		})
	})
	return imageUrls
}

// hasGalleryErrors checks if the gallery has any errors and should be skipped
func (m *ehentai) hasGalleryErrors(item *models.TrackedItem, html string) bool {
	if strings.Contains(html, "There are newer versions of this gallery available") {
		m.Logger.Info("newer version of gallery available, updating uri of: " + item.URI)
		document, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
		newGalleryLinks := document.Find("#gnd > a")
		// slice to retrieve only the latest gallery
		newGalleryLinks = newGalleryLinks.Slice(newGalleryLinks.Length()-1, newGalleryLinks.Length())
		newGalleryLinks.Each(func(index int, row *goquery.Selection) {
			url, exists := row.Attr("href")
			if exists {
				m.DbIO.GetFirstOrCreateTrackedItem(url, m)
				m.Logger.Info("added gallery to tracked items: " + url)
			}
		})
		return true
	}
	if strings.Contains(html, "This gallery has been removed or is unavailable.") {
		return true
	}
	return false
}

// getDownloadQueueItem extract the direct image URL from the passed gallery item
func (m *ehentai) getDownloadQueueItem(item imageGalleryItem) models.DownloadQueueItem {
	response, _ := m.Session.Get(item.uri)
	document := m.Session.GetDocument(response)
	imageURL, _ := document.Find("img#img").Attr("src")
	return models.DownloadQueueItem{
		ItemID:      item.id,
		DownloadTag: item.galleryTitle,
		FileName:    m.galleryImageIndexPattern.FindStringSubmatch(item.id)[1] + "_" + m.GetFileName(imageURL),
		FileURI:     imageURL,
	}
}

// extractGalleryTitle extracts the gallery title from the passed HTML
func (m *ehentai) extractGalleryTitle(html string) (galleryTitle string) {
	document, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	galleryTitle = document.Find("div#gd2 > h1#gn").Text()
	return m.SanitizePath(galleryTitle, false)
}
