package fourchan

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/DaRealFreak/watcher-go/internal/models"
)

// parseThread parses thread searches
func (m *fourChan) parseThread(item *models.TrackedItem) error {
	threadPattern := regexp.MustCompile(`.*/(?P<BoardId>.*)/thread/(?P<ThreadID>.*)/`)
	if threadPattern.MatchString(item.URI) {
		boardID := threadPattern.FindStringSubmatch(item.URI)[1]
		threadID := threadPattern.FindStringSubmatch(item.URI)[2]

		chanUrl := fmt.Sprintf("https://boards.4chan.org/%s/thread/%s", boardID, threadID)
		archiveUrl := fmt.Sprintf("https://desuarchive.org/%s/thread/%s/", boardID, threadID)

		completeThread := false
		if res, _ := m.Session.GetClient().Get(chanUrl); res.StatusCode == 404 {
			// original thread doesn't exist anymore, mark thread as completed if downloaded the most current item
			completeThread = true
		}

		res, err := m.Session.Get(archiveUrl)
		if err != nil {
			return err
		}

		html, _ := m.Session.GetDocument(res).Html()
		threadTitle := m.getThreadTitle(html)
		contentUrls := m.getThreadContents(html)

		keys := make([]int, 0, len(contentUrls))
		for k := range contentUrls {
			keys = append(keys, k)
		}
		sort.Ints(keys)

		// will return 0 on error, so fine for us too
		currentItemID, _ := strconv.ParseInt(item.CurrentItem, 10, 64)
		var downloadQueue []models.DownloadQueueItem

		for _, itemID := range keys {
			if item.CurrentItem == "" || itemID > int(currentItemID) {
				downloadQueue = append(downloadQueue, models.DownloadQueueItem{
					ItemID:      strconv.Itoa(itemID),
					DownloadTag: fmt.Sprintf("%s (%s)", threadTitle, threadID),
					FileURI:     fmt.Sprintf(contentUrls[itemID]),
					FileName:    fmt.Sprintf(fmt.Sprintf("%d_%s", itemID, m.GetFileName(contentUrls[itemID]))),
				})
			}
		}

		if err = m.ProcessDownloadQueue(downloadQueue, item); err != nil {
			return err
		}

		// if no error occurred during the download and thread doesn't exist on 4chan anymore mark item as complete
		if completeThread {
			m.DbIO.ChangeTrackedItemCompleteStatus(item, true)
		}
	}

	return nil
}

func (m *fourChan) getThreadTitle(html string) (title string) {
	document, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	document.Find("article:first-of-type > header h2.post_title").Each(func(i int, titleTag *goquery.Selection) {
		title = titleTag.Text()
	})

	return m.SanitizePath(title, false)
}

func (m *fourChan) getThreadContents(html string) map[int]string {
	contentUrls := make(map[int]string)
	document, _ := goquery.NewDocumentFromReader(strings.NewReader(html))

	document.Find("article.thread[id]").Each(func(i int, articleTag *goquery.Selection) {
		articleIdString, _ := articleTag.Attr("id")
		articleId, _ := strconv.ParseInt(articleIdString, 10, 64)

		articleTag.Find("div.thread_image_box").First().Find("a.thread_image_link[href]").Each(func(i int, aTag *goquery.Selection) {
			contentUrl, _ := aTag.Attr("href")
			contentUrls[int(articleId)] = contentUrl
		})
	})

	document.Find("aside.posts > article.has_image[id]").Each(func(i int, articleTag *goquery.Selection) {
		articleIdString, _ := articleTag.Attr("id")
		articleId, _ := strconv.ParseInt(articleIdString, 10, 64)

		articleTag.Find("a.thread_image_link[href]").Each(func(i int, aTag *goquery.Selection) {
			contentUrl, _ := aTag.Attr("href")
			contentUrls[int(articleId)] = contentUrl
		})
	})

	return contentUrls
}
