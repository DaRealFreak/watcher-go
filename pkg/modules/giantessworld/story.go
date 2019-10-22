package giantessworld

import (
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/PuerkitoBio/goquery"
	"github.com/jaytaylor/html2text"
	"github.com/spf13/viper"
)

// parseStory parses single stories
func (m *giantessWorld) parseStory(item *models.TrackedItem) error {
	m.addChapterToItemURI(item)

	base, _ := url.Parse(item.URI)

	res, err := m.Session.Get(item.URI)
	if err != nil {
		return err
	}

	htmlContent, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if item.CurrentItem == "" {
		err = m.downloadChapter(htmlContent, item)
		if err != nil {
			return err
		}
	}

	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(string(htmlContent)))

	nextChapterTag := doc.Find("div#next > a.next[href]")
	if nextChapterTag.Length() > 0 {
		nextChapterURI, _ := nextChapterTag.Attr("href")
		nextChapterURL, _ := url.Parse(nextChapterURI)

		res, err := m.Session.Get(base.ResolveReference(nextChapterURL).String())
		if err != nil {
			return err
		}

		htmlContent, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}

		// download chapter updating the item and repeat the function for recursively going through story pages
		err = m.downloadChapter(htmlContent, item)
		if err != nil {
			return err
		}

		return m.parseStory(item)
	}

	return nil
}

// addChapterToItemURI updates the item URI to match the current item argument
func (m *giantessWorld) addChapterToItemURI(item *models.TrackedItem) {
	if item.CurrentItem != "" {
		itemURI, _ := url.Parse(item.URI)

		// parse existing fragments and override with passed values (required for token)
		fragments := itemURI.Query()
		fragments.Set("chapter", item.CurrentItem)
		itemURI.RawQuery = fragments.Encode()
		item.URI = itemURI.String()
	}
}

// downloadChapter extracts the chapter from the html content and updates the item
func (m *giantessWorld) downloadChapter(htmlContent []byte, item *models.TrackedItem) error {
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(string(htmlContent)))

	content, err := doc.Find("div#story").First().Html()
	if err != nil {
		return err
	}

	text, err := html2text.FromString(content)
	if err != nil {
		return err
	}

	// ensure download directory since we directly create the files
	m.Session.EnsureDownloadDirectory(
		path.Join(
			viper.GetString("download.directory"),
			m.Key(),
			m.getAuthor(doc),
			"tmp.txt",
		),
	)

	filePath := path.Join(viper.GetString("download.directory"),
		m.Key(),
		m.getAuthor(doc),
		m.SanitizePath(m.getStoryName(doc)+"_"+m.getChapterTitle(doc)+".txt", false),
	)

	err = ioutil.WriteFile(filePath, []byte(text), os.ModePerm)
	if err != nil {
		return err
	}

	m.DbIO.UpdateTrackedItem(item, m.getChapterID(doc))

	return nil
}

// getAuthor extracts the author from the page title
func (m *giantessWorld) getAuthor(document *goquery.Document) string {
	author := document.Find("div#pagetitle > a[href*='viewuser.php']")
	return author.Text()
}

// getStoryTitle extracts the story name from the page title
func (m *giantessWorld) getStoryName(document *goquery.Document) string {
	story := document.Find("div#pagetitle > a[href*='viewstory.php']")
	return story.Text()
}

// getChapterTitle extracts the chapter title from the jump selection
func (m *giantessWorld) getChapterTitle(document *goquery.Document) string {
	selectedChapter := document.Find("form[name='jump'] > select > option[selected]")
	return selectedChapter.Text()
}

// getChapterID extracts the chapter ID from the jump selection
func (m *giantessWorld) getChapterID(document *goquery.Document) string {
	val, _ := document.Find("form[name='jump'] > select > option[selected]").Attr("value")
	return val
}
