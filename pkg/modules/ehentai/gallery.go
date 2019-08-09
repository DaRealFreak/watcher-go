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
	response, _ := m.Session.Get(item.Uri)
	html, _ := m.Session.GetDocument(response).Html()
	if m.hasGalleryErrors(item, html) {
		m.DbIO.ChangeTrackedItemCompleteStatus(item, true)
		return
	}
	galleryTitle := m.extractGalleryTitle(html)

	var downloadQueue []models.DownloadQueueItem
	foundCurrentItem := false
	response, _ = m.Session.Get(m.getLastGalleryPageUrl(html))
	html, _ = m.Session.GetDocument(response).Html()
	for foundCurrentItem == false {
		for _, galleryItem := range m.getGalleryImageUrls(html, galleryTitle) {
			if galleryItem.id != item.CurrentItem {
				downloadQueueItem := m.getDownloadQueueItem(galleryItem)
				// check for limit
				if downloadQueueItem.FileUri == "https://exhentai.org/img/509.gif" ||
					downloadQueueItem.FileUri == "https://e-hentai.org/img/509.gif" {
					log.Info("download limit reached, skipping galleries from now on")
					m.downloadLimitReached = true
					foundCurrentItem = true
					break
				}
				downloadQueue = append(downloadQueue, m.getDownloadQueueItem(galleryItem))
			} else {
				foundCurrentItem = true
				break
			}
		}

		// break outer loop too if the current item got found
		if foundCurrentItem {
			break
		}

		previousPageUrl, exists := m.getPreviousGalleryPageUrl(html)
		if exists == false {
			// no previous page exists anymore, break here
			break
		}
		response, _ = m.Session.Get(previousPageUrl)
		html, _ = m.Session.GetDocument(response).Html()
	}

	// reverse download queue to download the oldest items first
	// and to have a point to start with on the next run (last page -> front page)
	downloadQueue = m.ReverseDownloadQueueItems(downloadQueue)
	m.ProcessDownloadQueue(downloadQueue, item)
	if m.downloadLimitReached == false {
		// mark item as complete since update doesn't affect old galleries
		// if download limit got reached we didn't reach the final image and don't set the gallery as complete
		m.DbIO.ChangeTrackedItemCompleteStatus(item, true)
	}
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
	response, _ := m.Session.Get(item.uri)
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
