package schalenetwork

import (
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/models"
)

// parseSearch parses the tracked item if we detected a search or tag
func (m *schaleNetwork) parseSearch(item *models.TrackedItem) error {
	searchQuery, extraParams := m.extractSearchParams(item.URI)

	page := 1
	apiResponse, err := m.getSearch(searchQuery, page, extraParams)
	if err != nil {
		return err
	}

	if item.SubFolder == "" {
		m.DbIO.ChangeTrackedItemSubFolder(item, m.getSubFolder(item))
	}

	var itemQueue []searchEntry
	foundCurrentItem := false

	for {
		for _, entry := range apiResponse.Entries {
			currentItemID, _ := strconv.ParseInt(item.CurrentItem, 10, 64)

			if item.CurrentItem != "" && int64(entry.ID) <= currentItemID {
				foundCurrentItem = true
				break
			}

			itemQueue = append(itemQueue, entry)
		}

		if foundCurrentItem {
			break
		}

		if apiResponse.Limit*apiResponse.Page >= apiResponse.Total {
			break
		}

		page++
		apiResponse, err = m.getSearch(searchQuery, page, extraParams)
		if err != nil {
			return err
		}
	}

	// reverse to add the oldest items first
	for i, j := 0, len(itemQueue)-1; i < j; i, j = i+1, j-1 {
		itemQueue[i], itemQueue[j] = itemQueue[j], itemQueue[i]
	}

	slog.Info(fmt.Sprintf("found %d new items for uri: %s", len(itemQueue), item.URI), "module", m.Key)

	for _, entry := range itemQueue {
		galleryURL := fmt.Sprintf("%s/g/%d/%s", m.siteBaseURL(), entry.ID, entry.Key)
		galleryItem := m.DbIO.GetFirstOrCreateTrackedItem(galleryURL, m.getSubFolder(item), m)
		if (m.Cfg.Run.Force || m.Cfg.Run.ResetProgress) && galleryItem.CurrentItem != "" {
			slog.Info(
				fmt.Sprintf("resetting progress for item %s (current id: %s)", galleryItem.URI, galleryItem.CurrentItem),
				"module", m.Key,
			)
			galleryItem.CurrentItem = ""
			m.DbIO.ChangeTrackedItemCompleteStatus(galleryItem, false)
			m.DbIO.UpdateTrackedItem(galleryItem, "")
		}
	}

	for index, entry := range itemQueue {
		galleryURL := fmt.Sprintf("%s/g/%d/%s", m.siteBaseURL(), entry.ID, entry.Key)

		slog.Info(fmt.Sprintf(
			"added gallery to tracked items: \"%s\", search item: \"%s\" (%0.2f%%)",
			galleryURL,
			item.URI,
			float64(index+1)/float64(len(itemQueue))*100,
		), "module", m.Key)

		galleryItem := m.DbIO.GetFirstOrCreateTrackedItem(galleryURL, m.getSubFolder(item), m)

		id := strconv.Itoa(entry.ID)
		detail, err := m.getBookDetail(id, entry.Key)
		if err != nil {
			slog.Warn(
				fmt.Sprintf("error occurred getting book detail for %s (%s), skipping", galleryURL, err.Error()),
				"module", item.Module,
			)
			continue
		}

		if err = m.parseGalleryFromDetail(galleryItem, detail, id, entry.Key); err != nil {
			slog.Warn(
				fmt.Sprintf("error occurred parsing item %s (%s), skipping", galleryItem.URI, err.Error()),
				"module", item.Module,
			)
			return err
		}

		m.DbIO.UpdateTrackedItem(item, strconv.Itoa(entry.ID))
	}

	return nil
}

// extractSearchQuery extracts the search query from a URI
// extractSearchParams extracts the search query and any extra query parameters (like lang) from a URI
func (m *schaleNetwork) extractSearchParams(uri string) (string, url.Values) {
	extra := url.Values{}

	// check for browse/search URI with ?s= parameter
	browsePattern := regexp.MustCompile(`/browse\b`)
	if browsePattern.MatchString(uri) {
		parsedURL, parseErr := url.Parse(uri)
		if parseErr == nil && parsedURL.Query().Has("s") {
			query := parsedURL.Query().Get("s")
			// preserve other query parameters (e.g. lang)
			for key, values := range parsedURL.Query() {
				if key != "s" && key != "page" && key != "sort" {
					extra[key] = values
				}
			}

			return query, extra
		}
	}

	// check for tag URI
	matches := m.tagPattern.FindStringSubmatch(uri)
	if len(matches) > 1 {
		tag := matches[1]
		// URL decode the tag
		if decoded, decodeErr := url.QueryUnescape(tag); decodeErr == nil {
			tag = decoded
		}
		// replace + with space (same as gallery-dl)
		tag = strings.ReplaceAll(tag, "+", " ")
		// wrap tag value in ^...$ for exact matching (e.g. "artist:azuki" -> "artist:^azuki$")
		if parts := strings.SplitN(tag, ":", 2); len(parts) == 2 {
			tag = parts[0] + ":^" + parts[1] + "$"
		}
		return tag, extra
	}

	return "", extra
}

// getSubFolder returns the subfolder for categorization
func (m *schaleNetwork) getSubFolder(item *models.TrackedItem) string {
	browsePattern := regexp.MustCompile(`/browse\b`)
	if browsePattern.MatchString(item.URI) {
		parsedURL, parseErr := url.Parse(item.URI)
		if parseErr == nil && parsedURL.Query().Has("s") {
			query := parsedURL.Query().Get("s")
			query = strings.NewReplacer("^", "", "$", "").Replace(query)
			return fmt.Sprintf("search %s", query)
		}
	}

	matches := m.tagPattern.FindStringSubmatch(item.URI)
	if len(matches) > 1 {
		tag := matches[1]
		if decoded, decodeErr := url.QueryUnescape(tag); decodeErr == nil {
			tag = decoded
		}
		tag = strings.ReplaceAll(tag, "+", " ")
		// split namespace:tag into "namespace tag"
		parts := strings.SplitN(tag, ":", 2)
		if len(parts) == 2 {
			return fmt.Sprintf("%s %s", parts[0], parts[1])
		}
		return tag
	}

	return ""
}
