package ehentai

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/http"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
	"github.com/PuerkitoBio/goquery"
	log "github.com/sirupsen/logrus"
)

// imageGalleryItem contains the relevant data of gallery items
type imageGalleryItem struct {
	id           string
	uri          string
	galleryTitle string
}

// parseGallery parses the tracked item if we detected a tracked gallery
func (m *ehentai) parseGallery(item *models.TrackedItem) error {
	response, err := m.Session.Get(item.URI)
	if err != nil {
		return err
	}

	html, _ := m.Session.GetDocument(response).Html()
	if hasGalleryError, newGalleryItem := m.hasGalleryErrors(item, html); hasGalleryError {
		m.DbIO.ChangeTrackedItemCompleteStatus(item, true)
		if newGalleryItem != nil {
			return m.Parse(newGalleryItem)
		}

		return fmt.Errorf("gallery contains errors")
	}

	var downloadQueue []*imageGalleryItem

	galleryTitle := m.extractGalleryTitle(html)

	// check for exclusively allowed tags (if f.e. multiple languages are together in one gallery)
	whitelisted := false
	for _, tag := range m.settings.Search.WhitelistedTags {
		if strings.Contains(strings.ToLower(galleryTitle), strings.ToLower(tag)) {
			whitelisted = true
			break
		}
	}

	// if not exclusively allowed check for blacklisted tags
	if !whitelisted {
		for _, blacklistedTag := range m.settings.Search.BlacklistedTags {
			if strings.Contains(strings.ToLower(galleryTitle), strings.ToLower(blacklistedTag)) {
				log.WithField("module", m.Key).Warnf(
					"gallery title \"%s\" contains blacklisted tag \"%s\", setting item to complete",
					galleryTitle,
					blacklistedTag,
				)
				m.DbIO.ChangeTrackedItemCompleteStatus(item, true)
				return nil
			}
		}
	}

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

		// change to next proxy to avoid IP ban
		err = m.setProxyMethod()
		if err != nil {
			return err
		}

		response, _ = m.Session.Get(nextPageURL)
		html, _ = m.Session.GetDocument(response).Html()
	}

	if m.settings.MultiProxy {
		// reset usage and errors from previous galleries
		m.resetProxies()
		if err = m.processDownloadQueueMultiProxy(downloadQueue, item); err != nil {
			return err
		}
	} else {
		if err = m.processDownloadQueue(downloadQueue, item); err != nil {
			return err
		}
	}

	if !m.downloadLimitReached {
		// mark item as complete since update doesn't affect old galleries
		// if download limit got reached we didn't reach the final image and don't set the gallery as complete
		m.DbIO.ChangeTrackedItemCompleteStatus(item, true)
	}

	return nil
}

// getNextGalleryPageURL retrieves the url of the next page if it exists
func (m *ehentai) getNextGalleryPageURL(html string) (url string, exists bool) {
	document, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	pages := document.Find("table.ptb td")
	// return empty url if we don't have any result due to f.e. removed galleries
	if pages.Length() == 0 {
		return "", false
	}

	nextPageElement := pages.Slice(pages.Length()-1, pages.Length())
	return nextPageElement.Find("a[href]").Attr("href")
}

// getGalleryImageUrls extracts all gallery image urls from the passed html
func (m *ehentai) getGalleryImageUrls(html string, galleryTitle string) []*imageGalleryItem {
	var imageUrls []*imageGalleryItem

	document, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	document.Find("div#gdt > div a[href]").Each(func(index int, row *goquery.Selection) {
		uri, _ := row.Attr("href")
		imageUrls = append(imageUrls, &imageGalleryItem{
			id:           m.galleryImageIDPattern.FindString(uri),
			uri:          uri,
			galleryTitle: galleryTitle,
		})
	})

	return imageUrls
}

// hasGalleryErrors checks if the gallery has any errors and should be skipped
func (m *ehentai) hasGalleryErrors(item *models.TrackedItem, html string) (bool, *models.TrackedItem) {
	if strings.Contains(html, "There are newer versions of this gallery available") {
		log.WithField("module", m.Key).Info("newer version of gallery available, updating uri of: " + item.URI)

		document, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
		newGalleryLinks := document.Find("#gnd > a")
		// slice to retrieve only the latest gallery
		var newGalleryItem *models.TrackedItem
		newGalleryLinks = newGalleryLinks.Slice(newGalleryLinks.Length()-1, newGalleryLinks.Length())
		newGalleryLinks.Each(func(index int, row *goquery.Selection) {
			url, exists := row.Attr("href")
			if exists {
				newGalleryItem = m.DbIO.GetFirstOrCreateTrackedItem(url, m.getSubFolder(item), m)
				log.WithField("module", m.Key).Info("added gallery to tracked items: " + url)
			}
		})

		return true, newGalleryItem
	}

	if strings.Contains(html, "document.location = \"https://exhentai.org/\";") {
		log.WithField("module", m.Key).Warning("this gallery has been removed due to a copyright claim")
		return true, nil
	}

	return false, nil
}

// getDownloadQueueItem extract the direct image URL from the passed gallery item
func (m *ehentai) getDownloadQueueItem(
	downloadSession http.SessionInterface, trackedItem *models.TrackedItem, item *imageGalleryItem,
) (*models.DownloadQueueItem, error) {
	response, err := downloadSession.Get(item.uri)
	if err != nil {
		return nil, err
	}

	document := downloadSession.GetDocument(response)
	imageTag := document.Find("img#img")
	imageURL, _ := imageTag.Attr("src")

	downloadQueueItem := &models.DownloadQueueItem{
		ItemID: item.id,
		DownloadTag: fmt.Sprintf(
			"%s (%s)",
			item.galleryTitle,
			m.searchGalleryIDPattern.FindStringSubmatch(trackedItem.URI)[1],
		),
		FileName: m.galleryImageIndexPattern.FindStringSubmatch(item.id)[1] + "_" + fp.GetFileName(imageURL),
		FileURI:  imageURL,
	}

	if onError, exists := imageTag.Attr("onerror"); exists {
		fallbackRegexp := regexp.MustCompile(`this.onerror=null; nl\('(\d+-\d+)'\)`)
		if fallbackRegexp.MatchString(onError) {
			matches := fallbackRegexp.FindStringSubmatch(onError)
			if len(matches) == 2 {
				downloadQueueItem.FallbackFileURI = item.uri + "?nl=" + matches[1]
			}
		}
	}

	return downloadQueueItem, nil
}

// extractGalleryTitle extracts the gallery title from the passed HTML
func (m *ehentai) extractGalleryTitle(html string) (galleryTitle string) {
	document, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	galleryTitle = document.Find("div#gd2 > h1#gn").Text()

	return fp.SanitizePath(galleryTitle, false)
}
