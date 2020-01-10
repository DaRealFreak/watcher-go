package giantessworld

import (
	"net/url"
	"time"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/PuerkitoBio/goquery"
	log "github.com/sirupsen/logrus"
)

// storyMetaData contains the required meta data for the module
type storyMetaData struct {
	chapterURL    string
	chapterUpdate string
}

// parseUser parses all pages and adds them to the database
func (m *giantessWorld) parseUser(item *models.TrackedItem) error {
	var newStories []storyMetaData

	if item.CurrentItem == "" {
		// if no current item is set, set last update time to timestamp 0 for parsing
		item.CurrentItem = "January 01 1970"
	}

	foundCurrent := false
	currentPageURI := item.URI

	lastUpdate, err := time.Parse("January 02 2006", item.CurrentItem)
	if err != nil {
		return err
	}

	for !foundCurrent {
		currentPageURI = m.addSortingToURI(currentPageURI)

		res, err := m.Session.Get(currentPageURI)
		if err != nil {
			return err
		}

		doc := m.Session.GetDocument(res)
		for _, story := range m.extractStories(doc) {
			date, err := time.Parse("January 02 2006", story.chapterUpdate)
			if err != nil {
				return err
			}
			// item got updated after the date we are currently checking
			if date.After(lastUpdate) || date.Equal(lastUpdate) {
				newStories = append(newStories, story)
			} else {
				foundCurrent = true
				break
			}
		}

		if link, exists := doc.Find("div#pagelinks > a#plnext[href]").First().Attr("href"); exists {
			nextPageURI, err := url.Parse(link)
			if err != nil {
				return err
			}

			currentPageURI = m.baseURL.ResolveReference(nextPageURI).String()
		} else {
			break
		}
	}

	m.addNewStories(item, newStories)

	return nil
}

// extractStories extracts the story meta data from the passed document
func (m *giantessWorld) extractStories(doc *goquery.Document) (newStories []storyMetaData) {
	doc.Find("div.listbox").Each(func(i int, selection *goquery.Selection) {
		chapterURL, _ := selection.Find("a[href*='viewstory.php']").First().Attr("href")
		parsedChapterURL, _ := url.Parse(chapterURL)
		chapterUpdate := m.chapterUpdatePattern.FindStringSubmatch(selection.Text())[1]

		// item got updated after the date we are currently checking
		newStories = append(newStories, storyMetaData{
			chapterURL:    m.baseURL.ResolveReference(parsedChapterURL).String(),
			chapterUpdate: chapterUpdate,
		})
	})

	return newStories
}

// addSortingToURI adds the sort argument to the URI to ensure the sorting for continual progress
func (m *giantessWorld) addSortingToURI(itemURI string) string {
	itemURL, _ := url.Parse(itemURI)
	fragments := itemURL.Query()
	fragments.Set("sort", "update")
	itemURL.RawQuery = fragments.Encode()

	return itemURL.String()
}

// addNewStories adds the new stories to the database
func (m *giantessWorld) addNewStories(item *models.TrackedItem, newStories []storyMetaData) {
	// reverse stories to add the oldest stories first
	for i, j := 0, len(newStories)-1; i < j; i, j = i+1, j-1 {
		newStories[i], newStories[j] = newStories[j], newStories[i]
	}

	for _, story := range newStories {
		m.DbIO.UpdateTrackedItem(item, story.chapterUpdate)

		newItem := m.DbIO.GetFirstOrCreateTrackedItem(story.chapterURL, m)
		if newItem.CurrentItem == "" {
			// if story doesn't have a current item yet, it's probably a new story
			log.WithField("module", m.Key).Info("added story to tracked items: " + story.chapterURL)
		}
	}
}
