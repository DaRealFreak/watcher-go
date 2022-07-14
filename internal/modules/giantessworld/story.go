package giantessworld

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"strings"
	"unicode/utf8"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/PuerkitoBio/goquery"
	"github.com/jaytaylor/html2text"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// parseStory parses single stories
func (m *giantessWorld) parseStory(item *models.TrackedItem) error {
	base, _ := url.Parse(item.URI)

	htmlContent, err := m.getChapterContent(base, item.CurrentItem)
	if err != nil {
		return err
	}

	if bytes.Contains(htmlContent, []byte("Access denied. This story has not been validated by the adminstrators of this site.")) {
		log.WithField("module", m.Key).Warnf(
			"story has been deleted: \"%s\", setting item to completed",
			item.URI,
		)
		m.DbIO.ChangeTrackedItemCompleteStatus(item, true)

		return nil
	}

	if item.CurrentItem == "" {
		log.WithField("module", m.Key).Info(
			fmt.Sprintf(
				"downloading initial chapter for uri: \"%s\"",
				item.URI,
			),
		)

		err = m.downloadChapter(htmlContent, item)
		if err != nil {
			return err
		}
	}

	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(string(htmlContent)))
	newChapters := m.getNewChapters(doc)

	for index, chapter := range newChapters {
		htmlContent, err := m.getChapterContent(base, chapter)
		if err != nil {
			return err
		}

		// download chapter updating the item and repeat the function for recursively going through story pages
		log.WithField("module", m.Key).Info(
			fmt.Sprintf(
				"downloading updates for uri: \"%s\" (%0.2f%%)",
				item.URI,
				float64(index+1)/float64(len(newChapters))*100,
			),
		)

		err = m.downloadChapter(htmlContent, item)
		if err != nil {
			return err
		}
	}

	return nil
}

// getChapterContent builds the chapter URI from the base URL and returns the HTML content
func (m *giantessWorld) getChapterContent(base *url.URL, chapter string) (htmlContent []byte, err error) {
	fragments := base.Query()
	fragments.Set("chapter", chapter)
	base.RawQuery = fragments.Encode()

	res, err := m.Session.Get(base.String())
	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(res.Body)
}

// getNewChapters returns new chapters from the jump navigation
func (m *giantessWorld) getNewChapters(document *goquery.Document) []string {
	var results []string

	passedActiveSelection := false
	options := document.Find("form[name='jump'] > select > option")

	options.Each(func(i int, selection *goquery.Selection) {
		_, exists := selection.Attr("selected")
		if exists {
			passedActiveSelection = true
		}

		if !exists && passedActiveSelection {
			val, _ := selection.Attr("value")
			results = append(results, val)
		}
	})

	return results
}

// downloadChapter extracts the chapter from the html content and updates the item
func (m *giantessWorld) downloadChapter(htmlContent []byte, item *models.TrackedItem) error {
	parsedUrl, _ := url.Parse(item.URI)
	storyId, ok := parsedUrl.Query()["sid"]
	if !ok {
		return fmt.Errorf("unable to get story ID from item URI: %s", item.URI)
	}

	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(string(htmlContent)))

	content, err := doc.Find("div#story").First().Html()
	if err != nil {
		return err
	}

	text, err := html2text.FromString(content)
	if err != nil {
		return err
	}

	text = m.ensureUTF8(text)

	filePath := path.Join(viper.GetString("download.directory"),
		m.Key,
		m.getAuthor(doc),
		m.SanitizePath(storyId[0]+"_"+m.getStoryName(doc)+"_"+m.getChapterTitle(doc)+".txt", false),
	)

	// ensure download directory since we directly create the files
	m.Session.EnsureDownloadDirectory(filePath)

	err = ioutil.WriteFile(filePath, []byte(text), os.ModePerm)
	if err != nil {
		return err
	}

	m.Session.UpdateTreeFolderChangeTimes(filePath)
	m.DbIO.UpdateTrackedItem(item, m.getChapterID(doc))

	return nil
}

// getAuthor extracts the author from the page title
func (m *giantessWorld) getAuthor(document *goquery.Document) string {
	author := document.Find("div#pagetitle > a[href*='viewuser.php']")
	return strings.TrimSpace(m.ensureUTF8(author.Text()))
}

// getStoryTitle extracts the story name from the page title
func (m *giantessWorld) getStoryName(document *goquery.Document) string {
	story := document.Find("div#pagetitle > a[href*='viewstory.php']")
	return strings.TrimSpace(m.ensureUTF8(story.Text()))
}

// getChapterTitle extracts the chapter title from the jump selection
func (m *giantessWorld) getChapterTitle(document *goquery.Document) string {
	selectedChapter := document.Find("form[name='jump'] > select > option[selected]")
	return strings.TrimSpace(m.ensureUTF8(selectedChapter.Text()))
}

// getChapterID extracts the chapter ID from the jump selection
func (m *giantessWorld) getChapterID(document *goquery.Document) string {
	val, exists := document.Find("form[name='jump'] > select > option[selected]").Attr("value")
	if !exists {
		// one chapter story
		return "1"
	}

	return strings.TrimSpace(val)
}

func (m *giantessWorld) ensureUTF8(s string) string {
	fixUtf := func(r rune) rune {
		if r == utf8.RuneError {
			return -1
		}

		return r
	}

	return strings.Map(fixUtf, s)
}
