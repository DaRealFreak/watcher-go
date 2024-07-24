package kemono

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/PuerkitoBio/goquery"
)

type postItem struct {
	id    string
	title string
	uri   string
}

func (m *kemono) parseUser(item *models.TrackedItem) error {
	// update base URL
	if strings.Contains(item.URI, "coomer.su") {
		m.baseUrl, _ = url.Parse("https://coomer.su")
	} else {
		m.baseUrl, _ = url.Parse("https://kemono.su")
	}

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

func (m *kemono) parsePost(item *models.TrackedItem) error {
	// update base URL
	if strings.Contains(item.URI, "coomer.su") {
		m.baseUrl, _ = url.Parse("https://coomer.su")
	} else {
		m.baseUrl, _ = url.Parse("https://kemono.su")
	}

	// extract 2nd number from example URL: https://kemono.su/patreon/user/551274/post/24446001
	postId := regexp.MustCompile(`.*/post/(\d+)`).FindStringSubmatch(item.URI)
	if len(postId) != 2 {
		return fmt.Errorf("could not extract post ID from URL: %s", item.URI)
	}

	return m.processDownloadQueue(item, []*postItem{{
		id:    postId[1],
		title: "",
		uri:   item.URI,
	}})
}

func (m *kemono) getPostsUrls(html string) (postItems []*postItem) {
	document, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	document.Find("article[data-user][data-id]").Each(func(index int, row *goquery.Selection) {
		if dataId, exists := row.Attr("data-id"); exists {
			uriTag := row.Find("a[href*=\"/post/\"]")
			uri, _ := uriTag.Attr("href")
			parsedUri, _ := url.Parse(uri)
			absUri := m.baseUrl.ResolveReference(parsedUri)
			titleTag := row.Find("header.post-card__header")
			title := strings.TrimSpace(strings.ReplaceAll(titleTag.Text(), "\n", ""))

			postItems = append(postItems, &postItem{
				id:    dataId,
				title: title,
				uri:   absUri.String(),
			})
		}
	})

	return postItems
}
