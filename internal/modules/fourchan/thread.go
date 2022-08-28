package fourchan

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
	"github.com/PuerkitoBio/goquery"
)

// parseThread parses thread searches
func (m *fourChan) parseThread(item *models.TrackedItem) error {
	if m.threadPattern.MatchString(item.URI) {
		boardID := m.threadPattern.FindStringSubmatch(item.URI)[1]
		threadID := m.threadPattern.FindStringSubmatch(item.URI)[2]

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
		for _, blacklistedTag := range m.settings.Search.BlacklistedTags {
			if strings.Contains(strings.ToLower(threadTitle), strings.ToLower(blacklistedTag)) {
				log.WithField("module", m.Key).Warnf(
					"thread title \"%s\" contains blacklisted tag \"%s\", setting item to complete",
					threadTitle,
					blacklistedTag,
				)
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
					FileURI:     fmt.Sprintf(contentUrls[itemID]),
					FileName:    fmt.Sprintf(fmt.Sprintf("%d_%s", itemID, fp.GetFileName(contentUrls[itemID]))),
				})
			}
		}

		downloadQueue = m.filterDownloadQueue(downloadQueue)

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

func (m *fourChan) filterDownloadQueue(downloadQueue []models.DownloadQueueItem) []models.DownloadQueueItem {
	log.WithField("module", m.Key).Infof(
		"checking availability of %d found items",
		len(downloadQueue),
	)

	// reset possible previous download queues
	m.multiThreading.filteredDownloadQueue = []models.DownloadQueueItem{}

	// limit to 4 routines at the same time
	guard := make(chan struct{}, 4)
	for _, downloadQueueItem := range downloadQueue {
		guard <- struct{}{}
		m.multiThreading.waitGroup.Add(1)
		go func(downloadQueueItem models.DownloadQueueItem) {
			m.checkDownloadQueueItem(downloadQueueItem)
			<-guard
			m.multiThreading.waitGroup.Done()
		}(downloadQueueItem)
	}

	m.multiThreading.waitGroup.Wait()

	return m.multiThreading.filteredDownloadQueue
}

func (m *fourChan) checkDownloadQueueItem(downloadQueueItem models.DownloadQueueItem) {
	if res, _ := m.Session.GetClient().Get(downloadQueueItem.FileURI); res.StatusCode == 404 {
		log.WithField("module", m.Key).Debugf(
			"received 404 status code trying to reach url \"%s\", skipping item",
			downloadQueueItem.FileURI,
		)
		return
	}

	m.multiThreading.filteredDownloadQueue = append(m.multiThreading.filteredDownloadQueue, downloadQueueItem)
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
