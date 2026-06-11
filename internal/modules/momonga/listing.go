package momonga

import (
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/PuerkitoBio/goquery"
)

// listingItem is a gallery discovered on a listing page
type listingItem struct {
	id  string
	uri string
}

// extractListingGalleries returns the deduplicated gallery links found on a listing page
func (m *momonga) extractListingGalleries(html string) []listingItem {
	var items []listingItem
	seen := make(map[string]bool)

	document, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	document.Find("div.post-list a[href]").Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}

		matches := m.galleryPattern.FindStringSubmatch(href)
		if len(matches) != 2 {
			return
		}

		id := matches[1]
		if seen[id] {
			return
		}
		seen[id] = true

		items = append(items, listingItem{id: id, uri: href})
	})

	return items
}

// getNextListingPageURL returns the URL of the next listing page if it exists
func (m *momonga) getNextListingPageURL(html string) (string, bool) {
	document, _ := goquery.NewDocumentFromReader(strings.NewReader(html))

	return document.Find("a.nextpostslink[href]").First().Attr("href")
}

// parseListing paginates a listing page, discovers galleries, and parses each new one
func (m *momonga) parseListing(item *models.TrackedItem) error {
	res, err := m.get(item.URI)
	if err != nil {
		return err
	}

	if m.settings.Search.CategorizeSearch && item.SubFolder == "" {
		m.DbIO.ChangeTrackedItemSubFolder(item, m.getSubFolder(item))
	}

	html, _ := m.Session.GetDocument(res).Html()

	var itemQueue []listingItem
	foundCurrentItem := false

	// will return 0 on error, so fine for us too
	currentItemID, _ := strconv.ParseInt(item.CurrentItem, 10, 64)

	for {
		for _, listing := range m.extractListingGalleries(html) {
			galleryItemID, parseErr := strconv.ParseInt(listing.id, 10, 64)
			if parseErr != nil {
				return parseErr
			}

			if item.CurrentItem != "" && galleryItemID <= currentItemID {
				foundCurrentItem = true
				break
			}

			itemQueue = append(itemQueue, listing)
		}

		if foundCurrentItem {
			break
		}

		nextPageURL, exists := m.getNextListingPageURL(html)
		if !exists {
			break
		}

		// rotate proxy between pages to reduce the chance of an IP block
		if err = m.setProxyMethod(); err != nil {
			return err
		}

		res, err = m.get(nextPageURL)
		if err != nil {
			return err
		}

		html, _ = m.Session.GetDocument(res).Html()
	}

	// reverse to add the oldest items first so the run can be interrupted safely
	for i, j := 0, len(itemQueue)-1; i < j; i, j = i+1, j-1 {
		itemQueue[i], itemQueue[j] = itemQueue[j], itemQueue[i]
	}

	slog.Info(fmt.Sprintf("found %d new items for uri: %s", len(itemQueue), item.URI), "module", m.Key)

	// reset progress on discovered galleries when forced
	for _, listing := range itemQueue {
		galleryItem := m.DbIO.GetFirstOrCreateTrackedItem(listing.uri, m.getSubFolder(item), m)
		if (m.Cfg.Run.Force || m.Cfg.Run.ResetProgress) && galleryItem.CurrentItem != "" {
			slog.Info(fmt.Sprintf("resetting progress for item %s (current id: %s)",
				galleryItem.URI, galleryItem.CurrentItem), "module", m.Key)
			galleryItem.CurrentItem = ""
			m.DbIO.ChangeTrackedItemCompleteStatus(galleryItem, false)
			m.DbIO.UpdateTrackedItem(galleryItem, "")
		}
	}

	for index, listing := range itemQueue {
		galleryItem := m.DbIO.GetFirstOrCreateTrackedItem(listing.uri, m.getSubFolder(item), m)
		if !galleryItem.Complete {
			slog.Info(fmt.Sprintf(
				"added gallery to tracked items: \"%s\", search item: \"%s\" (%0.2f%%)",
				listing.uri,
				item.URI,
				float64(index+1)/float64(len(itemQueue))*100,
			), "module", m.Key)

			if err = m.Parse(galleryItem); err != nil {
				slog.Warn(fmt.Sprintf("error occurred parsing item %s (%s), skipping",
					galleryItem.URI, err.Error()), "module", m.Key)
				return err
			}
		}

		m.DbIO.UpdateTrackedItem(item, listing.id)
	}

	return nil
}

// getSubFolder derives a categorization subfolder from the listing type and slug
func (m *momonga) getSubFolder(item *models.TrackedItem) string {
	if !m.settings.Search.CategorizeSearch {
		return ""
	}

	if m.settings.Search.InheritSubFolder && item.SubFolder != "" {
		return item.SubFolder
	}

	parsed, err := url.Parse(item.URI)
	if err != nil {
		return ""
	}

	segments := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(segments) == 0 || segments[0] == "" {
		return ""
	}

	listingType := segments[0]
	switch listingType {
	case "cartoonist", "group", "character", "parody", "tag":
		if len(segments) >= 2 {
			slug, decodeErr := url.PathUnescape(segments[1])
			if decodeErr != nil {
				slug = segments[1]
			}
			return fmt.Sprintf("%s %s", listingType, slug)
		}
		return listingType
	case "trend", "popularity", "rated", "fanzine", "magazine":
		return listingType
	default:
		return ""
	}
}
