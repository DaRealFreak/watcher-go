package jinjamodoki

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/PuerkitoBio/goquery"
)

// parsePage parses a user page for contributions
func (m *jinjaModoki) parsePage(item *models.TrackedItem) error {
	var (
		downloadQueue []models.DownloadQueueItem
		err           error
	)

	foundCurrent := false
	currentPageURI := item.URI
	currentItemID, _ := strconv.ParseInt(item.CurrentItem, 10, 64)

	for !foundCurrent {
		res, err := m.Session.Get(currentPageURI)
		if err != nil {
			return err
		}

		doc := m.Session.GetDocument(res)
		currentPageOffset := m.getNavigationOffset(currentPageURI)

		doc.Find(`table.list > tbody > tr[class]`).Each(func(i int, selection *goquery.Selection) {
			if !foundCurrent {
				if downloadQueueItem, err := m.parseItem(selection); err == nil {
					itemID, _ := strconv.ParseInt(downloadQueueItem.ItemID, 10, 64)
					if itemID <= currentItemID {
						foundCurrent = true
						return
					}
					downloadQueue = append(downloadQueue, downloadQueueItem)
				}
			}
		})

		nextPage := m.findNextPage(doc)
		if !foundCurrent && nextPage.Length() > 0 {
			link, _ := nextPage.First().Attr("href")

			nextPageRelativeURI, err := url.Parse(link)
			if err != nil {
				return err
			}

			currentPageURI = m.baseURL.ResolveReference(nextPageRelativeURI).String()
			nextPageOffset := m.getNavigationOffset(currentPageURI)

			// the latest navigation link is the previous link, so we are done here
			if nextPageOffset <= currentPageOffset {
				break
			}
		}
	}

	for i, j := 0, len(downloadQueue)-1; i < j; i, j = i+1, j-1 {
		downloadQueue[i], downloadQueue[j] = downloadQueue[j], downloadQueue[i]
		downloadQueue[i].DownloadTag, err = m.getDownloadTagFromItemURI(item.URI)

		if err != nil {
			return err
		}
	}

	return m.processDownloadQueue(downloadQueue, item)
}

func (m *jinjaModoki) parseItem(selection *goquery.Selection) (downloadItem models.DownloadQueueItem, err error) {
	link := selection.Find(`td:nth-child(2) > a[href*="?file="]`).First()
	id := selection.Find(`td:nth-child(4) > input[name="did[]"][value]`).First()
	restrictions := selection.Find(`td:nth-child(10)`).First()
	relativeURI, _ := link.Attr("href")

	u, err := url.Parse(relativeURI)
	if err != nil {
		return downloadItem, err
	}

	if strings.TrimSpace(restrictions.Text()) != "" {
		res, err := m.Session.Get(m.baseURL.ResolveReference(u).String())
		if err != nil {
			return downloadItem, err
		}

		link = m.Session.GetDocument(res).Find(`table.info tr:nth-child(3) a[href*="/documents/"]`).First()
		relativeURI, _ = link.Attr("href")

		u, err = url.Parse(relativeURI)
		if err != nil {
			return downloadItem, err
		}
	}

	downloadItem.ItemID, _ = id.Attr("value")
	downloadItem.FileURI = m.baseURL.ResolveReference(u).String()
	downloadItem.FileName, err = m.getFileNameFromFileURI(downloadItem.ItemID, downloadItem.FileURI)

	return downloadItem, err
}

// findNextPage returns the HTML selection to the next page if existing
func (m *jinjaModoki) findNextPage(doc *goquery.Document) *goquery.Selection {
	return doc.Find("div.list_all > form a[href*='offset']:last-child")
}

// getNavigationOffset returns the navigation offset from the passed URI GET parameters (0 if not set)
func (m *jinjaModoki) getNavigationOffset(navigationURI string) int {
	u, _ := url.Parse(navigationURI)

	q, _ := url.ParseQuery(u.RawQuery)
	if len(q["doffset"]) == 0 {
		return 0
	}

	// on error it would return 0 too, which is our default if not set anyways
	offset, _ := strconv.ParseInt(q["doffset"][0], 10, 64)

	return int(offset)
}

func (m *jinjaModoki) getFileNameFromFileURI(itemID string, fileURI string) (string, error) {
	u, _ := url.Parse(fileURI)

	q, _ := url.ParseQuery(u.RawQuery)
	if len(q["file"]) == 0 {
		return "", fmt.Errorf("parsed uri(%s) does not contain any \"file\" tag", fileURI)
	}

	return fmt.Sprintf("%s_%s", itemID, filepath.Base(q["file"][0])), nil
}

func (m *jinjaModoki) getDownloadTagFromItemURI(fileURI string) (string, error) {
	u, _ := url.Parse(fileURI)

	q, _ := url.ParseQuery(u.RawQuery)
	if len(q["uname"]) == 0 {
		return "", fmt.Errorf("parsed uri(%s) does not contain any \"uname\" tag", fileURI)
	}

	return filepath.Base(q["uname"][0]), nil
}
