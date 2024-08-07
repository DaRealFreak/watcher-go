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
	var creator string
	creatorMatches := m.patterns.fanboxPattern.FindStringSubmatch(item.URI)
	if creatorMatches[1] != "" {
		creator = creatorMatches[1]
	} else {
		creator = creatorMatches[2]
	}

	var downloadQueue []*downloadQueueItem

	currentItemID, _ := strconv.ParseInt(item.CurrentItem, 10, 64)
	foundCurrentItem := false

	pagination, err := m.fanboxAPI.GetPostPagination(creator)
	if err != nil {
		return err
	}

	for _, paginationUrl := range pagination.URLs {
		if foundCurrentItem {
			break
		}

		postList, postListErr := m.fanboxAPI.GetPostListByURL(paginationUrl)
		if postListErr != nil {
			return postListErr
		}

		for _, fanboxPost := range postList.Body {
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
