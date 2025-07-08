package ehentai

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
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

// getSkCookie checks for the cookie "sk", since the search results are missing results without it
// the cookie only gets set in a response of a gallery
func (m *ehentai) getSkCookie(item *models.TrackedItem) (exists bool, value *http.Cookie) {
	requestUrl, _ := url.Parse(item.URI)
	cookies := m.Session.GetCookies(requestUrl)
	for _, cookie := range cookies {
		if cookie.Name == "sk" {
			return true, cookie
		}
	}

	return false, nil
}

// parseSearch parses the tracked item if we detected a search/tag
func (m *ehentai) parseSearch(item *models.TrackedItem) error {
	searchUrl, searchErr := m.getSearchURL(item)
	if searchErr != nil {
		return searchErr
	}

	if exists, _ := m.getSkCookie(item); !exists {
		// call an example gallery, which will return a "Set-Cookie" header containing the required sk cookie
		_, err := m.get("https://exhentai.org/g/1717239/a8f9b0c99c/")
		if err != nil {
			return err
		}

		existsNow, value := m.getSkCookie(item)
		if existsNow {
			m.DbIO.GetFirstOrCreateCookie("sk", value.Value, "", m)
		}
	}

	response, err := m.get(searchUrl)
	if err != nil {
		return err
	}

	if m.settings.Search.CategorizeSearch && item.SubFolder == "" {
		m.DbIO.ChangeTrackedItemSubFolder(item, m.getSubFolder(item))
	}

	var itemQueue []searchGalleryItem

	html, _ := m.Session.GetDocument(response).Html()
	foundCurrentItem := false

	for {
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

			if item.CurrentItem != "" && galleryItemID <= currentItemID {
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
			// no further page exists anymore, break here
			break
		}

		// change to next proxy to avoid IP ban
		err = m.setProxyMethod()
		if err != nil {
			return err
		}

		response, err = m.get(nextPageURL)
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

	for _, gallery := range itemQueue {
		galleryItem := m.DbIO.GetFirstOrCreateTrackedItem(gallery.uri, m.getSubFolder(item), m)
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
		galleryItem := m.DbIO.GetFirstOrCreateTrackedItem(gallery.uri, m.getSubFolder(item), m)
		if m.ipBanned {
			log.WithField("module", m.Key).Info(
				fmt.Sprintf(
					"download limit reached, skipping galleries for search item: \"%s\" (%0.2f%%)",
					item.URI,
					float64(index+1)/float64(len(itemQueue))*100,
				),
			)
			break
		}

		if !galleryItem.Complete {
			log.WithField("module", m.Key).Info(
				fmt.Sprintf(
					"added gallery to tracked items: \"%s\", search item: \"%s\" (%0.2f%%)",
					gallery.uri,
					item.URI,
					float64(index+1)/float64(len(itemQueue))*100,
				),
			)

			if err = m.Parse(galleryItem); err != nil {
				log.WithField("module", item.Module).Warningf(
					"error occurred parsing item %s (%s), skipping", galleryItem.URI, err.Error(),
				)
				return err
			}
		}

		m.DbIO.UpdateTrackedItem(item, gallery.id)
	}

	return nil
}

func (m *ehentai) getSubFolder(item *models.TrackedItem) string {
	// don't categorize searches, so return empty string
	if !m.settings.Search.CategorizeSearch {
		return ""
	}

	// if we inherit the sub folder from the search item
	// and the sub folder isn't empty return it instead of searching everytime
	if m.settings.Search.InheritSubFolder && item.SubFolder != "" {
		return item.SubFolder
	}

	search := regexp.MustCompile(`https://exhentai.org/.*f_search=.*`)
	if search.MatchString(item.URI) {
		parsedUrl, _ := url.Parse(item.URI)
		if parsedUrl.Query().Has("f_search") {
			return fmt.Sprintf("search %s", parsedUrl.Query().Get("f_search"))
		}
	}

	// if search was not matching we only have the tag options left
	folder := regexp.MustCompile(`https://exhentai.org/tag/(\w+):([^/?&]+).*`)
	if folder.MatchString(item.URI) {
		results := folder.FindStringSubmatch(item.URI)
		if len(results) == 3 {
			return fmt.Sprintf("%s %s", results[1], results[2])
		}
	}

	tag := regexp.MustCompile(`https://exhentai.org/tag/(.*)`)
	if tag.MatchString(item.URI) {
		results := tag.FindStringSubmatch(item.URI)
		if len(results) == 2 {
			return fmt.Sprintf("tag %s", results[1])
		}
	}

	// no matches at all
	return ""
}

// getSearchGalleryUrls returns all gallery URLs from the passed HTML
func (m *ehentai) getSearchGalleryUrls(html string) []searchGalleryItem {
	var items []searchGalleryItem

	document, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	document.Find(".itg [class*='gl3'] a[href]").Each(func(index int, row *goquery.Selection) {
		uri, _ := row.Attr("href")
		items = append(items, searchGalleryItem{
			id:  m.searchGalleryIDPattern.FindStringSubmatch(uri)[1],
			uri: uri,
		})
	})

	return items
}

// getNextSearchPageURL retrieves the url of the next page if it exists
func (m *ehentai) getNextSearchPageURL(html string) (url string, exists bool) {
	document, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	pages := document.Find("div.searchnav a#dnext")
	// return empty url if we don't have any result due to f.e. removed galleries
	if pages.Length() == 0 {
		return "", false
	}

	return pages.First().Attr("href")
}

func (m *ehentai) getSearchURL(item *models.TrackedItem) (string, error) {
	parsed, parsedErr := url.Parse(item.URI)
	if parsedErr != nil {
		return "", parsedErr
	}

	queries := parsed.Query()
	queries.Set("advsearch", "1")
	// search gallery name
	queries.Set("f_sname", "on")
	// search gallery tags
	queries.Set("f_stags", "on")

	if m.settings.Search.ExpungedGalleries {
		// search expunged galleries
		queries.Set("f_sh", "on")
	}

	if m.settings.Search.LowPoweredTags {
		// search low powered tags
		queries.Set("f_sdt1", "on")
	}

	parsed.RawQuery = queries.Encode()

	return parsed.String(), nil
}
