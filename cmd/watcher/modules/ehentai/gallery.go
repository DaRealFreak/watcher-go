package ehentai

import (
	"github.com/PuerkitoBio/goquery"
	"log"
	"strings"
	"watcher-go/cmd/watcher/models"
)

type imageGalleryItem struct {
	id           string
	uri          string
	galleryTitle string
}

func (m *ehentai) parseGallery(item *models.TrackedItem) []models.DownloadQueueItem {
	response, _ := m.Session.Get(item.Uri, 0)
	html, _ := m.Session.GetDocument(response).Html()
	if strings.Contains(html, "There are newer versions of this gallery available") {
		log.Fatal("newer version available, update tracked item for uri: " + item.Uri)
	}
	galleryTitle := m.extractGalleryTitle(html)

	var downloadQueue []models.DownloadQueueItem
	foundCurrentItem := false
	response, _ = m.Session.Get(m.getLastGalleryPageUrl(html), 0)
	html, _ = m.Session.GetDocument(response).Html()
	for foundCurrentItem == false {
		for _, galleryItem := range m.getGalleryImageUrls(html, galleryTitle) {
			if galleryItem.id != item.CurrentItem {
				downloadQueue = append(downloadQueue, m.getDownloadQueueItem(galleryItem))
			} else {
				foundCurrentItem = true
				break
			}
		}

		previousPageUrl, exists := m.getPreviousGalleryPageUrl(html)
		if !exists {
			// no previous page exists anymore, break here
			break
		}
		response, _ = m.Session.Get(previousPageUrl, 0)
		html, _ = m.Session.GetDocument(response).Html()
	}
	return downloadQueue
}

func (m *ehentai) getPreviousGalleryPageUrl(html string) (uri string, exists bool) {
	document, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	test := document.Find("table.ptb td").Slice(0, 1)
	return test.Find("a[href]").Attr("href")
}

// retrieve the last page since we have a limited amount of requests we can do
// so instead of extracting the start and not being able to extract file uris at the end, start at the end
func (m *ehentai) getLastGalleryPageUrl(html string) string {
	document, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	pages := document.Find("table.ptb td")
	lastPage := pages.Slice(pages.Length()-2, pages.Length()-1)
	lastPageUri, _ := lastPage.Find("a[href]").Attr("href")
	return lastPageUri
}

func (m *ehentai) getGalleryImageUrls(html string, galleryTitle string) []imageGalleryItem {
	var imageUrls []imageGalleryItem
	document, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	document.Find(".gdtm > div").Each(func(index int, row *goquery.Selection) {
		uri, _ := row.Find("a[href]").Attr("href")
		imageUrls = append(imageUrls, imageGalleryItem{
			id:           m.galleryImageIdPattern.FindString(uri),
			uri:          uri,
			galleryTitle: galleryTitle,
		})
	})

	// reverse so we check the oldest item first
	for i, j := 0, len(imageUrls)-1; i < j; i, j = i+1, j-1 {
		imageUrls[i], imageUrls[j] = imageUrls[j], imageUrls[i]
	}
	return imageUrls
}

func (m *ehentai) getDownloadQueueItem(item imageGalleryItem) models.DownloadQueueItem {
	response, _ := m.Session.Get(item.uri, 0)
	document := m.Session.GetDocument(response)
	imageUrl, _ := document.Find("img#img").Attr("src")
	return models.DownloadQueueItem{
		ItemId:      item.id,
		DownloadTag: item.galleryTitle,
		FileName:    m.GetFileName(imageUrl),
		FileUri:     imageUrl,
	}
}

func (m *ehentai) extractGalleryTitle(html string) (galleryTitle string) {
	document, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	galleryTitle = document.Find("div#gd2 > h1#gn").Text()
	return m.SanitizePath(galleryTitle, false)
}
