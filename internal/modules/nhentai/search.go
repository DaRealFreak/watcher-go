package nhentai

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/PuerkitoBio/goquery"
	log "github.com/sirupsen/logrus"
)

// searchGalleryItem contains the required variables for gallery items of the search function
type searchGalleryItem struct {
	id  string
	uri string
}

// parseSearch parses the tracked item if we detected a search/tag
func (m *nhentai) parseSearch(item *models.TrackedItem) error {
	response, err := m.Session.Get(item.URI)
	if err != nil {
		return err
	}

	var itemQueue []searchGalleryItem

	html, _ := m.Session.GetDocument(response).Html()
	foundCurrentItem := false

	for !foundCurrentItem {
		for _, galleryItem := range m.getSearchGalleryUrls(html) {
			var (
				currentItemID int64
				galleryItemID int64
			)

			// will return 0 on error, so fine for us too for the current item
			currentItemID, _ = strconv.ParseInt(item.CurrentItem, 10, 64)
			galleryItemID, err = strconv.ParseInt(galleryItem.id, 10, 64)
			if err != nil {
				return err
			}

			if !(item.CurrentItem == "" || galleryItemID > currentItemID) {
				foundCurrentItem = true
				break
			}

			itemQueue = append(itemQueue, galleryItem)
		}

		// break outer loop too if the current item got found
		if foundCurrentItem {
			break
		}

		nextPageURL, exists := m.getNextSearchPageURL(html)
		if !exists {
			// no next page exists anymore, break here
			break
		}

		response, err = m.Session.Get(nextPageURL)
		if err != nil {
			return err
		}

		html, _ = m.Session.GetDocument(response).Html()
	}

	// reverse to add the oldest items first
	for i, j := 0, len(itemQueue)-1; i < j; i, j = i+1, j-1 {
		itemQueue[i], itemQueue[j] = itemQueue[j], itemQueue[i]
	}

	log.WithField("module", m.Key).Info(
		fmt.Sprintf("found %d new items for uri: %s", len(itemQueue), item.URI),
	)

	// add items
	for index, gallery := range itemQueue {
		log.WithField("module", m.Key).Info(
			fmt.Sprintf(
				"added gallery to tracked items: \"%s\" (%0.2f%%)",
				gallery.uri,
				float64(index+1)/float64(len(itemQueue))*100,
			),
		)

		galleryItem := m.DbIO.GetFirstOrCreateTrackedItem(gallery.uri, m)
		if m.Cfg.Run.ForceNew && galleryItem.CurrentItem != "" {
			log.WithField("module", m.Key).Info(
				fmt.Sprintf("resetting progress for item %s (current id: %s)", galleryItem.URI, galleryItem.CurrentItem),
			)
			galleryItem.CurrentItem = ""
			m.DbIO.ChangeTrackedItemCompleteStatus(item, false)
			m.DbIO.UpdateTrackedItem(item, "")
		}

		if err = m.Parse(galleryItem); err != nil {
			log.WithField("module", item.Module).Warningf(
				"error occurred parsing item %s (%s), skipping", galleryItem.URI, err.Error(),
			)
			continue
		}

		m.DbIO.UpdateTrackedItem(item, gallery.id)
	}

	return nil
}

// getSearchGalleryUrls returns all gallery URLs from the passed HTML
func (m *nhentai) getSearchGalleryUrls(html string) []searchGalleryItem {
	var items []searchGalleryItem

	document, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	document.Find("div.index-container > div.gallery > a[href*=\"/g/\"]").Each(func(index int, row *goquery.Selection) {
		uri, _ := row.Attr("href")

		items = append(items, searchGalleryItem{
			id:  m.searchGalleryIDPattern.FindStringSubmatch(uri)[1],
			uri: m.getAbsoluteUri(uri),
		})
	})

	return items
}

// getNextSearchPageURL retrieves the url of the next page if it exists
func (m *nhentai) getNextSearchPageURL(html string) (string, bool) {
	document, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	pages := document.Find("section.pagination > a.next[href]")
	// return empty url if we don't have any result due to f.e. removed galleries
	if pages.Length() == 0 {
		return "", false
	}

	nextPageUrl, _ := pages.First().Attr("href")
	return m.getAbsoluteUri(nextPageUrl), true
}
