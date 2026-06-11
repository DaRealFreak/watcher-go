package momonga

import (
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
	"github.com/PuerkitoBio/goquery"
)

// galleryImage is one page image of a gallery
type galleryImage struct {
	page int
	uri  string
}

// extractGalleryTitle extracts the gallery title from the passed HTML
func (m *momonga) extractGalleryTitle(html string) string {
	document, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	title := document.Find("h1").First().Text()

	return fp.SanitizePath(strings.TrimSpace(title), false)
}

// extractGalleryImages extracts all page images from the #post-hentai container,
// keeping only direct gallery CDN URLs and sorting them by page number
func (m *momonga) extractGalleryImages(html string) []galleryImage {
	var images []galleryImage

	document, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	document.Find("div#post-hentai img[src]").Each(func(_ int, s *goquery.Selection) {
		src, exists := s.Attr("src")
		if !exists {
			return
		}

		matches := m.imagePattern.FindStringSubmatch(src)
		if len(matches) != 2 {
			return
		}

		page, convErr := strconv.Atoi(matches[1])
		if convErr != nil {
			return
		}

		images = append(images, galleryImage{page: page, uri: src})
	})

	sort.Slice(images, func(i, j int) bool {
		return images[i].page < images[j].page
	})

	return images
}

// extractContentTags returns the structured content tags of a gallery page
func (m *momonga) extractContentTags(html string) []string {
	var tags []string

	document, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	document.Find("div#post-tag a[rel=tag]").Each(func(_ int, s *goquery.Selection) {
		if text := strings.TrimSpace(s.Text()); text != "" {
			tags = append(tags, text)
		}
	})

	return tags
}

// isBlacklisted checks the gallery title and its content tags against the configured blacklist.
// Returns the matched blacklist term for logging.
func (m *momonga) isBlacklisted(html string, title string) (bool, string) {
	if len(m.settings.Search.BlacklistedTags) == 0 {
		return false, ""
	}

	haystacks := []string{strings.ToLower(title)}
	for _, tag := range m.extractContentTags(html) {
		haystacks = append(haystacks, strings.ToLower(tag))
	}

	for _, blacklisted := range m.settings.Search.BlacklistedTags {
		needle := strings.ToLower(blacklisted)
		for _, haystack := range haystacks {
			if strings.Contains(haystack, needle) {
				return true, blacklisted
			}
		}
	}

	return false, ""
}

// parseGallery parses a tracked gallery item and downloads its new page images
func (m *momonga) parseGallery(item *models.TrackedItem) error {
	res, err := m.get(item.URI)
	if err != nil {
		return err
	}

	html, _ := m.Session.GetDocument(res).Html()

	title := m.extractGalleryTitle(html)

	if !m.Cfg.Run.Force {
		if blacklisted, term := m.isBlacklisted(html, title); blacklisted {
			slog.Warn(fmt.Sprintf(
				"gallery \"%s\" contains blacklisted tag \"%s\", setting item to complete",
				title, term), "module", m.Key)
			m.DbIO.ChangeTrackedItemCompleteStatus(item, true)
			return nil
		}
	}

	galleryID := ""
	if matches := m.galleryPattern.FindStringSubmatch(item.URI); len(matches) == 2 {
		galleryID = matches[1]
	}

	// will return 0 on error, so fine for us too
	currentItem, _ := strconv.Atoi(item.CurrentItem)

	var downloadQueue []models.DownloadQueueItem
	for _, img := range m.extractGalleryImages(html) {
		if item.CurrentItem == "" || img.page > currentItem {
			downloadQueue = append(downloadQueue, models.DownloadQueueItem{
				ItemID:      strconv.Itoa(img.page),
				DownloadTag: fmt.Sprintf("%s (%s)", title, galleryID),
				FileURI:     img.uri,
				FileName:    fmt.Sprintf("%04d%s", img.page, fp.GetFileExtension(img.uri)),
			})
		}
	}

	// a fresh gallery that yielded no images likely indicates a markup/parsing change;
	// surface it rather than silently marking the item complete with nothing downloaded
	if len(downloadQueue) == 0 && item.CurrentItem == "" {
		slog.Warn(fmt.Sprintf("gallery \"%s\" yielded no images, not marking complete", title), "module", m.Key)
		return nil
	}

	if m.settings.MultiProxy {
		// reset usage and errors from previous galleries
		m.resetProxies()
		if err = m.processDownloadQueueMultiProxy(downloadQueue, item); err != nil {
			return err
		}
	} else {
		if err = m.ProcessDownloadQueue(downloadQueue, item); err != nil {
			return err
		}
	}

	// galleries are static finished works, mark complete once fully downloaded
	m.DbIO.ChangeTrackedItemCompleteStatus(item, true)

	return nil
}
