package pixiv

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/DaRealFreak/watcher-go/internal/models"
	mobileapi "github.com/DaRealFreak/watcher-go/internal/modules/pixiv/mobile_api"
	log "github.com/sirupsen/logrus"
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

	response, err := m.mobileAPI.GetSearchIllust(searchWord, searchMode, mobileapi.SearchOrderDateDescending, 0)
	if err != nil {
		return err
	}

	for !foundCurrentItem {
		for _, illustration := range response.Illustrations {
			if item.CurrentItem == "" || illustration.ID > int(currentItemID) {
				downloadQueue = append(downloadQueue, &downloadQueueItem{
					ItemID:       illustration.ID,
					DownloadTag:  m.SanitizePath(searchWord, false),
					DownloadItem: illustration,
				})
			} else {
				foundCurrentItem = true
				break
			}
		}

		if response.NextURL == "" {
			break
		}

		response, err = m.mobileAPI.GetSearchIllustByURL(response.NextURL)
		if err != nil {
			if err.Error() == `{"offset":["Offset must be no more than 5000"]}` {
				log.WithField("module", m.Key).Warningf(
					"search \"%s\" has more than 5000 results."+
						"The mobile API is limited to 5000 search results, if you want to download "+
						"more than 5000 results consider switching to the public search API",
					searchWord,
				)

				break
			}

			return err
		}
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
