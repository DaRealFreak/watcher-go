package pixiv

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/DaRealFreak/watcher-go/pkg/models"
	mobileapi "github.com/DaRealFreak/watcher-go/pkg/modules/pixiv/mobile_api"
	publicapi "github.com/DaRealFreak/watcher-go/pkg/modules/pixiv/public_api"
)

func (m *pixiv) parseSearch(item *models.TrackedItem) error {
	return nil
}

func (m *pixiv) parseSearchPublic(item *models.TrackedItem) error {
	searchWord := m.patterns.searchPattern.FindStringSubmatch(item.URI)[1]

	searchMode, err := m.getSearchModeFromURI(item.URI)
	if err != nil {
		return err
	}

	var downloadQueue []*downloadQueueItem

	currentItemID, _ := strconv.ParseInt(item.CurrentItem, 10, 64)
	foundCurrentItem := false
	page := 1

	for !foundCurrentItem {
		response, err := m.publicAPI.GetSearchIllust(
			searchWord,
			searchMode,
			publicapi.SearchOrderDescending,
			page,
		)
		if err != nil {
			return err
		}

		for _, publicIllustration := range response.Illustrations {
			if item.CurrentItem == "" || publicIllustration.ID > int(currentItemID) {
				downloadQueue = append(downloadQueue, &downloadQueueItem{
					ItemID:       publicIllustration.ID,
					DownloadTag:  m.SanitizePath(searchWord, false),
					DownloadItem: publicIllustration,
				})
			} else {
				foundCurrentItem = true
				break
			}
		}

		if response.Pagination.Next != nil {
			page = *response.Pagination.Next
		} else {
			break
		}
	}

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
		if m.settings.SearchAPI == SearchAPIPublic {
			return publicapi.SearchModeExactTagMatch, nil
		}

		return mobileapi.SearchModeExactTagMatch, nil
	case len(q["s_mode"]) > 0 && q["s_mode"][0] == "s_tag":
		if m.settings.SearchAPI == SearchAPIPublic {
			return publicapi.SearchModePartialTagMatch, nil
		}

		return mobileapi.SearchModePartialTagMatch, nil
	case len(q["s_mode"]) > 0 && q["s_mode"][0] == "s_tc":
		if m.settings.SearchAPI == SearchAPIPublic {
			return publicapi.SearchModeTitleAndCaption, nil
		}

		return mobileapi.SearchModeTitleAndCaption, nil
	default:
		return "", fmt.Errorf("unknown search mode used: %s", q["s_mode"][0])
	}
}
