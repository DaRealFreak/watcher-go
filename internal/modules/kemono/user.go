package kemono

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/PuerkitoBio/goquery"
)

type postItem struct {
	id  string
	uri string
}

func (m *kemono) parseUser(item *models.TrackedItem) error {
	response, err := m.Session.Get(item.URI)
	if err != nil {
		return err
	}

	html, _ := m.Session.GetDocument(response).Html()

	var (
		downloadQueue    []*postItem
		foundCurrentItem bool
		offset           int
	)

	for {
		posts := m.getPostsUrls(html)

		// we are beyond the last page, break here
		if len(posts) == 0 {
			break
		}

		for _, post := range posts {
			// check if we reached the current item already
			if post.id == item.CurrentItem {
				foundCurrentItem = true
				break
			}

			downloadQueue = append(downloadQueue, post)
		}

		if foundCurrentItem {
			break
		}

		// increase offset for the next page
		offset += 50
		pageUrl, _ := url.Parse(item.URI)
		queries := pageUrl.Query()
		queries.Set("o", strconv.Itoa(offset))
		pageUrl.RawQuery = queries.Encode()

		response, _ = m.Session.Get(pageUrl.String())
		// new behavior of kemono.su is to redirect to the main page if the offset exceeds the last page
		if response.Request.URL.String() != pageUrl.String() {
			break
		}

		html, _ = m.Session.GetDocument(response).Html()
	}

	// reverse download queue to download old items first
	for i, j := 0, len(downloadQueue)-1; i < j; i, j = i+1, j-1 {
		downloadQueue[i], downloadQueue[j] = downloadQueue[j], downloadQueue[i]
	}

	return m.processDownloadQueue(item, downloadQueue)
}

func (m *kemono) getPostsUrls(html string) (postItems []*postItem) {
	document, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	document.Find("article[data-user][data-id]").Each(func(index int, row *goquery.Selection) {
		if dataId, exists := row.Attr("data-id"); exists {
			uriTag := row.Find("a[href*=\"/post/\"]")
			uri, _ := uriTag.Attr("href")
			parsedUri, _ := url.Parse(uri)
			absUri := m.baseUrl.ResolveReference(parsedUri)

			postItems = append(postItems, &postItem{
				id:  dataId,
				uri: absUri.String(),
			})
		}
	})

	return postItems
}
