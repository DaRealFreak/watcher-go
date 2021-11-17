package nhentai

import (
	"fmt"
	"net/url"
	"strconv"
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
	galleryItems := m.getGalleryImageUrls(html, galleryTitle)

	// iterate backwards over the items to be able to break on the first match
	for i := len(galleryItems) - 1; i >= 0; i-- {
		galleryItem := galleryItems[i]

		// found current item, breaking
		if item.CurrentItem == galleryItem.ItemID {
			break
		}

		downloadQueue = append(downloadQueue, galleryItem)
	}

	// reverse download queue to download new items first
	for i, j := 0, len(downloadQueue)-1; i < j; i, j = i+1, j-1 {
		downloadQueue[i], downloadQueue[j] = downloadQueue[j], downloadQueue[i]
	}

	if err = m.ProcessDownloadQueue(downloadQueue, item); err != nil {
		return err
	}

	return nil
}

func (m *nhentai) getGalleryImageUrls(html string, title string) (galleryItems []models.DownloadQueueItem) {
	document, _ := goquery.NewDocumentFromReader(strings.NewReader(html))

	document.Find("div#thumbnail-container > div.thumbs > div.thumb-container img.lazyload").Each(
		func(i int, imageTag *goquery.Selection) {
			dataSrc, exists := imageTag.Attr("data-src")
			if !exists {
				return
			}

			imageUri, err := url.Parse(dataSrc)
			if err != nil {
				return
			}

			// replace thumbnail host with image host and convert thumb path segment to image path
			imageUri.Host = "i.nhentai.net"
			imageUri.Path = m.thumbToImageRegexp.ReplaceAllString(imageUri.Path, "$1$2")

			galleryItems = append(galleryItems, models.DownloadQueueItem{
				ItemID: strconv.Itoa(i + 1),
				DownloadTag: fmt.Sprintf(
					"%s (%s)",
					title,
					m.galleryIDPattern.FindStringSubmatch(imageUri.String())[1],
				),
				FileName: fmt.Sprintf("%d_%s", i+1, m.GetFileName(imageUri.String())),
				FileURI:  imageUri.String(),
			})
		})

	return galleryItems
}

// extractGalleryTitle extracts the gallery title from the passed HTML
func (m *nhentai) extractGalleryTitle(html string) (galleryTitle string) {
	document, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	galleryTitle = document.Find("div#info > h1.title").Text()

	return m.SanitizePath(galleryTitle, false)
}
