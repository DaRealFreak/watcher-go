package nhentai

import (
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/PuerkitoBio/goquery"
)

// parseGallery parses the tracked item if we detected a tracked gallery
func (m *nhentai) parseGallery(item *models.TrackedItem) error {
	response, err := m.Session.Get(item.URI)
	if err != nil {
		return err
	}

	html, _ := m.Session.GetDocument(response).Html()

	var downloadQueue []models.DownloadQueueItem

	galleryTitle := m.extractGalleryTitle(html)
	foundCurrentItem := item.CurrentItem == ""

	for _, currentGalleryItem := range m.getGalleryImageUrls(html, galleryTitle) {
		if foundCurrentItem {
			downloadQueue = append(downloadQueue, currentGalleryItem)
		}
		// check if we reached the current item already
		if currentGalleryItem.ItemID == item.CurrentItem {
			foundCurrentItem = true
		}
	}

	if err = m.ProcessDownloadQueue(downloadQueue, item); err != nil {
		return err
	}

	return nil
}

func (m *nhentai) getGalleryImageUrls(html string, title string) (galleryItems []models.DownloadQueueItem) {
	return galleryItems
}

// extractGalleryTitle extracts the gallery title from the passed HTML
func (m *nhentai) extractGalleryTitle(html string) (galleryTitle string) {
	document, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	galleryTitle = document.Find("div#gd2 > h1#gn").Text()

	return m.SanitizePath(galleryTitle, false)
}
