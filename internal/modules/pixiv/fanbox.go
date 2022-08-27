package pixiv

import (
	"fmt"
	"path"
	"strconv"

	"github.com/DaRealFreak/watcher-go/internal/models"
	fanboxapi "github.com/DaRealFreak/watcher-go/internal/modules/pixiv/fanbox_api"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
)

func (m *pixiv) parseFanbox(item *models.TrackedItem) error {
	creator := m.patterns.fanboxPattern.FindStringSubmatch(item.URI)[1]

	var downloadQueue []*downloadQueueItem

	currentItemID, _ := strconv.ParseInt(item.CurrentItem, 10, 64)
	foundCurrentItem := false

	postList, err := m.fanboxAPI.GetPostList(creator, nil, 0, 200)
	if err != nil {
		return err
	}

	for !foundCurrentItem {
		for _, fanboxPost := range postList.Body.Items {
			postID, _ := fanboxPost.ID.Int64()

			if item.CurrentItem == "" || postID != currentItemID {
				downloadQueue = append(downloadQueue, &downloadQueueItem{
					ItemID:       int(postID),
					DownloadTag:  path.Join(m.getFanboxDownloadTag(item, fanboxPost.User), "fanbox"),
					DownloadItem: fanboxPost,
				})
			} else {
				foundCurrentItem = true
				break
			}
		}

		if postList.Body.NextURL == "" {
			break
		}

		postList, err = m.fanboxAPI.GetPostListByURL(postList.Body.NextURL)
		if err != nil {
			return err
		}
	}

	// reverse download queue to download old items first
	for i, j := 0, len(downloadQueue)-1; i < j; i, j = i+1, j-1 {
		downloadQueue[i], downloadQueue[j] = downloadQueue[j], downloadQueue[i]
	}

	return m.processDownloadQueue(downloadQueue, item)
}

func (m *pixiv) getFanboxDownloadTag(item *models.TrackedItem, user fanboxapi.FanboxUser) string {
	if m.settings.UseSubFolderAsUsername && item.SubFolder != "" {
		return fmt.Sprintf(
			"%s/%s",
			user.UserID,
			fp.TruncateMaxLength(fp.SanitizePath(item.SubFolder, false)),
		)
	} else {
		return user.GetUserTag()
	}
}
