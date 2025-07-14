package nhentai

import (
	"fmt"
	"github.com/DaRealFreak/watcher-go/internal/models"
	log "github.com/sirupsen/logrus"
	"net/url"
	"regexp"
	"strconv"
)

// parseSearch parses the tracked item if we detected a search/tag
func (m *nhentai) parseSearch(item *models.TrackedItem) error {
	searchQuery := ""
	search := regexp.MustCompile(`https://nhentai.net/search/.*`)
	if search.MatchString(item.URI) {
		parsedUrl, _ := url.Parse(item.URI)
		if parsedUrl.Query().Has("q") {
			searchQuery = parsedUrl.Query().Get("q")
		}
	}

	// if search was not matching we only have those options left
	subFolderTags := []string{"artist", "tag", "parody", "character"}
	folder := regexp.MustCompile(`https://nhentai.net/\w+/([^/?&]+)/$`)
	for _, subFolder := range subFolderTags {
		subFolderRegexp := regexp.MustCompile(fmt.Sprintf(`https://nhentai.net/%s/.*`, subFolder))
		if subFolderRegexp.MatchString(item.URI) {
			matches := folder.FindStringSubmatch(item.URI)
			if len(matches) > 1 {
				searchQuery = fmt.Sprintf("%s:%s", subFolder, matches[1])
				break
			}
		}
	}

	page := 1
	apiResponse, err := m.getSearch(searchQuery, page, SortingRecent)
	if err != nil {
		return err
	}

	if m.settings.Search.CategorizeSearch && item.SubFolder == "" {
		m.DbIO.ChangeTrackedItemSubFolder(item, m.getSubFolder(item))
	}

	var itemQueue []*galleryResponse
	foundCurrentItem := false
	for {
		for _, gallery := range apiResponse.Galleries {
			// will return 0 on error, so fine for us too for the current item
			currentItemID, _ := strconv.ParseInt(item.CurrentItem, 10, 64)
			galleryItemID, _ := gallery.GalleryID.Int64()

			if item.CurrentItem != "" && galleryItemID <= currentItemID {
				foundCurrentItem = true
				break
			}

			itemQueue = append(itemQueue, gallery)
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
		apiResponse, err = m.getSearch(searchQuery, page, SortingRecent)
		if err != nil {
			return err
		}
	}

	// reverse to add the oldest items first
	for i, j := 0, len(itemQueue)-1; i < j; i, j = i+1, j-1 {
		itemQueue[i], itemQueue[j] = itemQueue[j], itemQueue[i]
	}

	log.WithField("module", m.Key).Info(
		fmt.Sprintf("found %d new items for uri: %s", len(itemQueue), item.URI),
	)

	for _, gallery := range itemQueue {
		galleryItem := m.DbIO.GetFirstOrCreateTrackedItem(gallery.GetURL(), m.getSubFolder(item), m)
		if (m.Cfg.Run.Force || m.Cfg.Run.ResetProgress) && galleryItem.CurrentItem != "" {
			log.WithField("module", m.Key).Info(
				fmt.Sprintf("resetting progress for item %s (current id: %s)", galleryItem.URI, galleryItem.CurrentItem),
			)
			galleryItem.CurrentItem = ""
			m.DbIO.ChangeTrackedItemCompleteStatus(galleryItem, false)
			m.DbIO.UpdateTrackedItem(galleryItem, "")
		}
	}

	// add items
	for index, gallery := range itemQueue {
		log.WithField("module", m.Key).Info(
			fmt.Sprintf(
				"added gallery to tracked items: \"%s\", search item: \"%s\" (%0.2f%%)",
				gallery.GetURL(),
				item.URI,
				float64(index+1)/float64(len(itemQueue))*100,
			),
		)

		galleryItem := m.DbIO.GetFirstOrCreateTrackedItem(gallery.GetURL(), m.getSubFolder(item), m)
		if err = m.parseGalleryFromApiResponse(galleryItem, gallery); err != nil {
			log.WithField("module", item.Module).Warningf(
				"error occurred parsing item %s (%s), skipping", galleryItem.URI, err.Error(),
			)
			return err
		}

		m.DbIO.UpdateTrackedItem(item, gallery.GalleryID.String())
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

	search := regexp.MustCompile(`https://nhentai.net/search/.*`)
	if search.MatchString(item.URI) {
		parsedUrl, _ := url.Parse(item.URI)
		if parsedUrl.Query().Has("q") {
			return fmt.Sprintf("search %s", parsedUrl.Query().Get("q"))
		}
	}

	// if search was not matching we only have those options left
	subFolderTags := []string{"artist", "tag", "parody", "character"}
	folder := regexp.MustCompile(`https://nhentai.net/\w+/([^/?&]+)/$`)

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
