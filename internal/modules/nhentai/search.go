package nhentai

import (
	"fmt"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"log/slog"
	"net/url"
	"regexp"
	"strconv"
)

// parseSearch parses the tracked item if we detected a search/tag
func (m *nhentai) parseSearch(item *models.TrackedItem) error {
	if m.settings.Search.CategorizeSearch && item.SubFolder == "" {
		m.DbIO.ChangeTrackedItemSubFolder(item, m.getSubFolder(item))
	}

	// check if this is a tag URL (artist/tag/parody/character)
	tagType, tagSlug := m.extractTagFromURL(item.URI)
	if tagType != "" && tagSlug != "" {
		return m.parseTagSearch(item, tagType, tagSlug)
	}

	// otherwise handle as a regular search
	return m.parseQuerySearch(item)
}

// extractTagFromURL extracts the tag type and slug from a tag URL
func (m *nhentai) extractTagFromURL(uri string) (string, string) {
	subFolderTags := []string{"artist", "tag", "parody", "character", "group", "language", "category"}
	folder := regexp.MustCompile(`https://nhentai.net/\w+/([^/?&]+)/?`)

	for _, tagType := range subFolderTags {
		subFolderRegexp := regexp.MustCompile(fmt.Sprintf(`https://nhentai.net/%s/.*`, tagType))
		if subFolderRegexp.MatchString(uri) {
			matches := folder.FindStringSubmatch(uri)
			if len(matches) > 1 {
				return tagType, matches[1]
			}
		}
	}

	return "", ""
}

// parseTagSearch handles tag-based URLs using the v2 tag API
func (m *nhentai) parseTagSearch(item *models.TrackedItem, tagType string, tagSlug string) error {
	tag, err := m.getTag(tagType, tagSlug)
	if err != nil {
		return err
	}

	tagID := tag.ID.String()

	page := 1
	apiResponse, err := m.getTaggedGalleries(tagID, page, SortingDate)
	if err != nil {
		return err
	}

	itemQueue, err := m.collectSearchResults(item, apiResponse, page, func(p int) (*searchResponse, error) {
		return m.getTaggedGalleries(tagID, p, SortingDate)
	})
	if err != nil {
		return err
	}

	return m.processSearchQueue(item, itemQueue)
}

// parseQuerySearch handles search URLs using the v2 search API
func (m *nhentai) parseQuerySearch(item *models.TrackedItem) error {
	searchQuery := ""
	search := regexp.MustCompile(`https://nhentai.net/search[/?].*`)
	if search.MatchString(item.URI) {
		parsedUrl, _ := url.Parse(item.URI)
		if parsedUrl.Query().Has("q") {
			searchQuery = parsedUrl.Query().Get("q")
		}
	}

	page := 1
	apiResponse, err := m.getSearch(searchQuery, page, SortingDate)
	if err != nil {
		return err
	}

	itemQueue, err := m.collectSearchResults(item, apiResponse, page, func(p int) (*searchResponse, error) {
		return m.getSearch(searchQuery, p, SortingDate)
	})
	if err != nil {
		return err
	}

	return m.processSearchQueue(item, itemQueue)
}

// collectSearchResults collects all new gallery IDs from paginated search results
func (m *nhentai) collectSearchResults(
	item *models.TrackedItem,
	apiResponse *searchResponse,
	page int,
	getNextPage func(int) (*searchResponse, error),
) ([]*searchResultItem, error) {
	var itemQueue []*searchResultItem
	foundCurrentItem := false

	for {
		for _, result := range apiResponse.Result {
			// will return 0 on error, so fine for us too for the current item
			currentItemID, _ := strconv.ParseInt(item.CurrentItem, 10, 64)
			resultItemID, _ := result.ID.Int64()

			if item.CurrentItem != "" && resultItemID <= currentItemID {
				foundCurrentItem = true
				break
			}

			itemQueue = append(itemQueue, result)
		}

		// break the outer loop too if the current item got found
		if foundCurrentItem {
			break
		}

		numberOfPages, _ := apiResponse.NumPages.Int64()
		if page >= int(numberOfPages) {
			// no further pages exist, break here
			break
		}

		page++
		var err error
		apiResponse, err = getNextPage(page)
		if err != nil {
			return nil, err
		}
	}

	// reverse to add the oldest items first
	for i, j := 0, len(itemQueue)-1; i < j; i, j = i+1, j-1 {
		itemQueue[i], itemQueue[j] = itemQueue[j], itemQueue[i]
	}

	return itemQueue, nil
}

// processSearchQueue processes the collected search results by fetching full gallery data
func (m *nhentai) processSearchQueue(item *models.TrackedItem, itemQueue []*searchResultItem) error {
	slog.Info(fmt.Sprintf("found %d new items for uri: %s", len(itemQueue), item.URI), "module", m.Key)

	for _, result := range itemQueue {
		galleryItem := m.DbIO.GetFirstOrCreateTrackedItem(result.GetURL(), m.getSubFolder(item), m)
		if (m.Cfg.Run.Force || m.Cfg.Run.ResetProgress) && galleryItem.CurrentItem != "" {
			slog.Info(fmt.Sprintf("resetting progress for item %s (current id: %s)", galleryItem.URI, galleryItem.CurrentItem), "module", m.Key)
			galleryItem.CurrentItem = ""
			m.DbIO.ChangeTrackedItemCompleteStatus(galleryItem, false)
			m.DbIO.UpdateTrackedItem(galleryItem, "")
		}
	}

	// add items
	for index, result := range itemQueue {
		slog.Info(fmt.Sprintf(
			"added gallery to tracked items: \"%s\", search item: \"%s\" (%0.2f%%)",
			result.GetURL(),
			item.URI,
			float64(index+1)/float64(len(itemQueue))*100,
		), "module", m.Key)

		galleryItem := m.DbIO.GetFirstOrCreateTrackedItem(result.GetURL(), m.getSubFolder(item), m)

		// fetch full gallery data for download
		fullGallery, err := m.getGallery(result.ID.String())
		if err != nil {
			slog.Warn(fmt.Sprintf("error occurred fetching gallery %s (%s), skipping", result.GetURL(), err.Error()), "module", item.Module)
			continue
		}

		if err = m.parseGalleryFromApiResponse(galleryItem, fullGallery); err != nil {
			slog.Warn(fmt.Sprintf("error occurred parsing item %s (%s), skipping", galleryItem.URI, err.Error()), "module", item.Module)
			return err
		}

		m.DbIO.UpdateTrackedItem(item, result.ID.String())
	}

	return nil
}

func (m *nhentai) getSubFolder(item *models.TrackedItem) string {
	// don't categorize searches, so return empty string
	if !m.settings.Search.CategorizeSearch {
		return ""
	}

	// if we inherit the subfolder from the search item
	// and the subfolder isn't empty, return it instead of searching everytime
	if m.settings.Search.InheritSubFolder && item.SubFolder != "" {
		return item.SubFolder
	}

	search := regexp.MustCompile(`https://nhentai.net/search[/?].*`)
	if search.MatchString(item.URI) {
		parsedUrl, _ := url.Parse(item.URI)
		if parsedUrl.Query().Has("q") {
			return fmt.Sprintf("search %s", parsedUrl.Query().Get("q"))
		}
	}

	// if search was not matching we only have those options left
	subFolderTags := []string{"artist", "tag", "parody", "character", "group", "language", "category"}
	folder := regexp.MustCompile(`https://nhentai.net/\w+/([^/?&]+)/?`)

	for _, subFolder := range subFolderTags {
		subFolderRegexp := regexp.MustCompile(fmt.Sprintf(`https://nhentai.net/%s/.*`, subFolder))
		if subFolderRegexp.MatchString(item.URI) {
			matches := folder.FindStringSubmatch(item.URI)
			if len(matches) > 1 {
				return fmt.Sprintf("%s %s", subFolder, matches[1])
			}
		}
	}

	// no matches at all
	return ""
}
