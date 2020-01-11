package jinjamodoki

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/PuerkitoBio/goquery"
)

// parsePage parses a user page for contributions
func (m *jinjaModoki) parsePage(item *models.TrackedItem) error {
	var downloadQueue []downloadQueueItem

	foundCurrent := false
	currentPageURI := item.URI
	downloadTag, err := m.getDownloadTagFromItemURI(item.URI)

	if err != nil {
		return err
	}

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
					downloadQueueItem.DownloadTag = downloadTag
					if downloadQueueItem.ItemID == item.CurrentItem {
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
		} else {
			break
		}
	}

	for i, j := 0, len(downloadQueue)-1; i < j; i, j = i+1, j-1 {
		downloadQueue[i], downloadQueue[j] = downloadQueue[j], downloadQueue[i]
	}

	return m.processDownloadQueue(downloadQueue, item)
}

func (m *jinjaModoki) parseItem(selection *goquery.Selection) (downloadItem downloadQueueItem, err error) {
	link := selection.Find(`td:nth-child(2) > a[href*="?file="]`).First()
	relativeURI, _ := link.Attr("href")
	restrictions := selection.Find(`td:last-child`).First()

	u, err := url.Parse(relativeURI)
	if err != nil {
		return downloadItem, err
	}

	downloadItem.FileURI = m.baseURL.ResolveReference(u).String()
	downloadItem.FileName, err = m.getFileNameFromFileURI(downloadItem.FileURI)
	downloadItem.ItemID = downloadItem.FileName
	downloadItem.restriction = strings.TrimSpace(restrictions.Text()) != ""

	return downloadItem, err
}

// findNextPage returns the HTML selection to the next page if existing
func (m *jinjaModoki) findNextPage(doc *goquery.Document) *goquery.Selection {
	return doc.Find(`div.list_all a[href*="offset"]:last-child`)
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

func (m *jinjaModoki) getFileNameFromFileURI(fileURI string) (string, error) {
	u, _ := url.Parse(fileURI)
	q, _ := url.ParseQuery(u.RawQuery)

	if len(q["file"]) > 0 {
		return filepath.Base(q["file"][0]), nil
	}

	if strings.Index(u.Path, "/documents/") == 0 {
		return filepath.Base(u.Path), nil
	}

	return "", fmt.Errorf("parsed uri(%s) does not contain any \"file\" tag", fileURI)
}

func (m *jinjaModoki) getDownloadTagFromItemURI(fileURI string) (string, error) {
	u, _ := url.Parse(fileURI)

	q, _ := url.ParseQuery(u.RawQuery)
	if len(q["uname"]) == 0 {
		return "", fmt.Errorf("parsed uri(%s) does not contain any \"uname\" tag", fileURI)
	}

	return filepath.Base(q["uname"][0]), nil
}
