package ehentai

import (
	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/PuerkitoBio/goquery"
	log "github.com/sirupsen/logrus"
	"strings"
)

type imageGalleryItem struct {
	id           string
	uri          string
	galleryTitle string
}

func (m *ehentai) parseGallery(item *models.TrackedItem) {
	response, _ := m.get(item.Uri)
	html, _ := m.Session.GetDocument(response).Html()
	if m.hasGalleryErrors(item, html) {
		m.DbIO.ChangeTrackedItemCompleteStatus(item, true)
		return
	}
	galleryTitle := m.extractGalleryTitle(html)

	var downloadQueue []imageGalleryItem
	foundCurrentItem := item.CurrentItem == ""

	for true {
		for _, galleryItem := range m.getGalleryImageUrls(html, galleryTitle) {
			if foundCurrentItem == true {
				downloadQueue = append(downloadQueue, galleryItem)
			}
			// check if we reached the current item already
			if galleryItem.id == item.CurrentItem {
				foundCurrentItem = true
			}
		}

		nextPageUrl, exists := m.getNextGalleryPageUrl(html)
		if exists == false {
			// no previous page exists anymore, break here
			break
		}
		response, _ = m.get(nextPageUrl)
		html, _ = m.Session.GetDocument(response).Html()
	}

	m.processDownloadQueue(downloadQueue, item)
	if m.downloadLimitReached == false {
		// mark item as complete since update doesn't affect old galleries
		// if download limit got reached we didn't reach the final image and don't set the gallery as complete
		m.DbIO.ChangeTrackedItemCompleteStatus(item, true)
	}
}

// retrieve url of the next page if it exists
func (m *ehentai) getNextGalleryPageUrl(html string) (url string, exists bool) {
	document, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	pages := document.Find("table.ptb td")
	nextPageElement := pages.Slice(pages.Length()-1, pages.Length())
	return nextPageElement.Find("a[href]").Attr("href")
}

// extract the gallery image urls from the passed html
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
	return imageUrls
}

// check if gallery has errors and should be skipped
func (m *ehentai) hasGalleryErrors(item *models.TrackedItem, html string) bool {
	if strings.Contains(html, "There are newer versions of this gallery available") {
		log.Info("newer version of gallery available, updating uri of: " + item.Uri)
		document, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
		newGalleryLinks := document.Find("#gnd > a")
		// slice to retrieve only the latest gallery
		newGalleryLinks = newGalleryLinks.Slice(newGalleryLinks.Length()-1, newGalleryLinks.Length())
		newGalleryLinks.Each(func(index int, row *goquery.Selection) {
			url, exists := row.Attr("href")
			if exists {
				m.DbIO.GetFirstOrCreateTrackedItem(url, m)
				log.Info("added gallery to tracked items: " + url)
			}
		})
		return true
	}
	if strings.Contains(html, "This gallery has been removed or is unavailable.") {
		return true
	}
	return false
}

func (m *ehentai) getDownloadQueueItem(item imageGalleryItem) models.DownloadQueueItem {
	response, _ := m.get(item.uri)
	document := m.Session.GetDocument(response)
	imageUrl, _ := document.Find("img#img").Attr("src")
	return models.DownloadQueueItem{
		ItemId:      item.id,
		DownloadTag: item.galleryTitle,
		FileName:    m.galleryImageIndexPattern.FindStringSubmatch(item.id)[1] + "_" + m.GetFileName(imageUrl),
		FileUri:     imageUrl,
	}
}

func (m *ehentai) extractGalleryTitle(html string) (galleryTitle string) {
	document, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	galleryTitle = document.Find("div#gd2 > h1#gn").Text()
	return m.SanitizePath(galleryTitle, false)
}
