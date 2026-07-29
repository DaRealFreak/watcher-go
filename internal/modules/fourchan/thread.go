package fourchan

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"log/slog"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
	"github.com/PuerkitoBio/goquery"
	fhttp "github.com/bogdanfinn/fhttp"
)

// parseThread parses thread searches
func (m *fourChan) parseThread(item *models.TrackedItem) error {
	if m.threadPattern.MatchString(item.URI) {
		boardID := m.threadPattern.FindStringSubmatch(item.URI)[1]
		threadID := m.threadPattern.FindStringSubmatch(item.URI)[2]

		chanUrl := fmt.Sprintf("https://boards.4chan.org/%s/thread/%s", boardID, threadID)
		archiveUrl := fmt.Sprintf("https://desuarchive.org/%s/thread/%s/", boardID, threadID)

		// original thread doesn't exist anymore, mark thread as completed if downloaded the most current item
		completeThread := m.threadRemoved(chanUrl)

		res, err := m.getPage(archiveUrl)
		if err != nil {
			return err
		}

		html, _ := m.Session.GetDocument(res).Html()
		threadTitle := m.getThreadTitle(html)
		for _, blacklistedTag := range m.settings.Search.BlacklistedTags {
			if strings.Contains(strings.ToLower(threadTitle), strings.ToLower(blacklistedTag)) {
				slog.Warn(fmt.Sprintf("thread title \"%s\" contains blacklisted tag \"%s\", setting item to complete",
					threadTitle,
					blacklistedTag), "module", m.Key)
				m.DbIO.ChangeTrackedItemCompleteStatus(item, true)
				return nil
			}
		}

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
					FileURI:     contentUrls[itemID],
					FileName:    fmt.Sprintf("%d_%s", itemID, fp.GetFileName(contentUrls[itemID])),
				})
			}
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

		// if no error occurred during the download and thread doesn't exist on 4chan anymore mark item as complete
		if completeThread {
			m.DbIO.ChangeTrackedItemCompleteStatus(item, true)
		}
	}

	return nil
}

// threadRemoved reports whether the original 4chan thread is gone (HTTP 404). It
// deliberately probes with the raw client to bypass the archive error handlers, since a
// 404 is an expected outcome here rather than a failure.
func (m *fourChan) threadRemoved(chanUrl string) bool {
	res, err := m.Session.GetClient().Get(chanUrl)
	if err != nil || res == nil {
		// a failed request returns a nil response, so we can neither dereference it nor
		// tell whether the thread is gone. treat it as still alive to keep a temporary
		// network problem from completing the tracked item.
		return false
	}

	if res.Body != nil {
		defer raven.CheckClosureNonFatal(res.Body)
	}

	return res.StatusCode == fhttp.StatusNotFound
}

func (m *fourChan) getThreadTitle(html string) (title string) {
	document, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	document.Find("article:first-of-type > header h2.post_title").Each(func(i int, titleTag *goquery.Selection) {
		title = titleTag.Text()
	})

	return fp.SanitizePath(title, false)
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
