package fourchan

import (
	"fmt"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/PuerkitoBio/goquery"
)

// parseSearch parses searches
func (m *fourChan) parseSearch(item *models.TrackedItem) error {
	res, err := m.Session.Get(item.URI)
	if err != nil {
		return err
	}

	html, _ := m.Session.GetDocument(res).Html()
	foundCurrentItem := false

	var (
		currentItemID int64
		galleryItemID int64
		itemQueue     []string
	)

	for !foundCurrentItem {
		threads := m.getThreads(html)
		for _, thread := range threads {
			threadID := m.threadPattern.FindStringSubmatch(thread)[2]

			// will return 0 on error, so fine for us too for the current item
			currentItemID, _ = strconv.ParseInt(item.CurrentItem, 10, 64)
			galleryItemID, err = strconv.ParseInt(threadID, 10, 64)

			if !(item.CurrentItem == "" || galleryItemID > currentItemID) {
				foundCurrentItem = true
				break
			}

			itemQueue = append(itemQueue, thread)
		}

		// break outer loop too if the current item got found
		if foundCurrentItem {
			break
		}

		nextPageURL, exists := m.getNextSearchPageURL(html)
		if !exists {
			// we're on the last page already, break here
			break
		}

		res, err = m.Session.Get(nextPageURL)
		if err != nil {
			return err
		}

		html, _ = m.Session.GetDocument(res).Html()
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
				gallery,
				float64(index+1)/float64(len(itemQueue))*100,
			),
		)

		galleryItem := m.DbIO.GetFirstOrCreateTrackedItem(gallery, "", m)
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

		threadID := m.threadPattern.FindStringSubmatch(gallery)[2]

		m.DbIO.UpdateTrackedItem(item, threadID)
	}

	return nil
}

func (m *fourChan) getThreads(html string) (threads []string) {
	document, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	document.Find("article > div.post_wrapper span.post_controls a[href*=\"/thread/\"]").Each(func(i int, titleTag *goquery.Selection) {
		if href, exists := titleTag.Attr("href"); exists {
			if m.threadPattern.MatchString(href) {
				href = strings.Split(href, "#")[0]
				threads = append(threads, href)
			}
		}
	})

	return threads
}

// getNextSearchPageURL retrieves the url of the next page if it exists
func (m *fourChan) getNextSearchPageURL(html string) (url string, exists bool) {
	document, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	pages := document.Find("div.paginate li.next > a[href*=\"/page/\"]")
	if pages.Length() == 0 {
		return "", false
	}

	return pages.First().Attr("href")
}
