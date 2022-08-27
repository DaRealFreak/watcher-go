package pixiv

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/DaRealFreak/watcher-go/internal/models"
	mobileapi "github.com/DaRealFreak/watcher-go/internal/modules/pixiv/mobile_api"
	pixivapi "github.com/DaRealFreak/watcher-go/internal/modules/pixiv/pixiv_api"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
)

func (m *pixiv) parseSearch(item *models.TrackedItem) error {
	searchWord := m.patterns.searchPattern.FindStringSubmatch(item.URI)[1]
	searchWord, _ = url.QueryUnescape(searchWord)

	searchMode, err := m.getSearchModeFromURI(item.URI)
	if err != nil {
		return err
	}

	var downloadQueue []*downloadQueueItem

	currentItemID, _ := strconv.ParseInt(item.CurrentItem, 10, 64)
	foundCurrentItem := false

	var (
		offset    int
		startDate *time.Time
		endDate   *time.Time
		response  *mobileapi.SearchIllust
	)

	for !foundCurrentItem {
		response, err = m.mobileAPI.GetSearchIllust(searchWord, searchMode, mobileapi.SearchOrderDateDescending, offset, 0, startDate, endDate)
		if err != nil {
			if _, ok := err.(pixivapi.OffsetError); ok {
				if len(downloadQueue) > 0 {
					lastIllustration := downloadQueue[len(downloadQueue)-1].DownloadItem.(mobileapi.Illustration)
					offset = 0
					endDate = &lastIllustration.CreateDate
					tmp := endDate.Add(-1 * 365 * 24 * time.Hour)
					startDate = &tmp
					continue
				}
			} else {
				return err
			}
		}

		for _, illustration := range response.Illustrations {
			if item.CurrentItem == "" || illustration.ID > int(currentItemID) {
				downloadQueue = append(downloadQueue, &downloadQueueItem{
					ItemID:       illustration.ID,
					DownloadTag:  fp.TruncateMaxLength(fp.SanitizePath(m.getDownloadTag(item), false)),
					DownloadItem: illustration,
				})
			} else {
				foundCurrentItem = true
				break
			}
		}

		if response.NextURL == "" && (startDate == nil || offset == 0) {
			break
		}

		// ToDo: parse response.NextURL for offset
		offset += 30
	}

	// reverse download queue to download old items first
	for i, j := 0, len(downloadQueue)-1; i < j; i, j = i+1, j-1 {
		downloadQueue[i], downloadQueue[j] = downloadQueue[j], downloadQueue[i]
	}

	return m.processDownloadQueue(downloadQueue, item)
}

func (m *pixiv) getSearchModeFromURI(searchURI string) (string, error) {
	u, _ := url.Parse(searchURI)
	q, _ := url.ParseQuery(u.RawQuery)

	switch {
	case len(q["s_mode"]) == 0 || q["s_mode"][0] == "s_tag_full":
		return mobileapi.SearchModeExactTagMatch, nil
	case len(q["s_mode"]) > 0 && q["s_mode"][0] == "s_tag":
		return mobileapi.SearchModePartialTagMatch, nil
	case len(q["s_mode"]) > 0 && q["s_mode"][0] == "s_tc":
		return mobileapi.SearchModeTitleAndCaption, nil
	default:
		return "", fmt.Errorf("unknown search mode used: %s", q["s_mode"][0])
	}
}

func (m *pixiv) getDownloadTag(item *models.TrackedItem) string {
	if item.SubFolder != "" {
		return item.SubFolder
	} else {
		searchWord := m.patterns.searchPattern.FindStringSubmatch(item.URI)[1]
		searchWord, _ = url.QueryUnescape(searchWord)
		return searchWord
	}
}
