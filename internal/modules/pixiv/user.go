package pixiv

import (
	"fmt"
	"strconv"

	"github.com/DaRealFreak/watcher-go/internal/models"
	mobileapi "github.com/DaRealFreak/watcher-go/internal/modules/pixiv/mobile_api"
	pixivapi "github.com/DaRealFreak/watcher-go/internal/modules/pixiv/pixiv_api"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
	log "github.com/sirupsen/logrus"
)

func (m *pixiv) parseUser(item *models.TrackedItem) error {
	userID, _ := strconv.ParseInt(m.patterns.memberPattern.FindStringSubmatch(item.URI)[1], 10, 64)

	userDetail, err := m.mobileAPI.GetUserDetail(int(userID))
	if err != nil {
		switch err.(type) {
		case pixivapi.UserUnavailableError:
			log.WithField("module", m.Key).Warningf(
				"couldn't retrieve user details, changing artist to complete (%s)",
				item.URI,
			)
			m.DbIO.ChangeTrackedItemCompleteStatus(item, true)

			return nil
		default:
			return err
		}
	}

	if m.settings.UseSubFolderAsUsername && item.SubFolder == "" {
		m.DbIO.ChangeTrackedItemSubFolder(item, userDetail.User.Name)
	}

	var downloadQueue []*downloadQueueItem

	currentItemID, _ := strconv.ParseInt(item.CurrentItem, 10, 64)
	foundCurrentItem := false

	response, err := m.mobileAPI.GetUserIllusts(int(userID), "", 0)
	if err != nil {
		return err
	}

	for !foundCurrentItem {
		for _, illustration := range response.Illustrations {
			if item.CurrentItem == "" || illustration.ID > int(currentItemID) {
				downloadQueue = append(downloadQueue, &downloadQueueItem{
					ItemID:       illustration.ID,
					DownloadTag:  m.getIllustDownloadTag(item, illustration.User),
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

		response, err = m.mobileAPI.GetUserIllustsByURL(response.NextURL)
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

func (m *pixiv) getIllustDownloadTag(item *models.TrackedItem, user mobileapi.UserInfo) string {
	if m.settings.UseSubFolderAsUsername && item.SubFolder != "" {
		return fmt.Sprintf(
			"%d/%s",
			user.ID,
			fp.TruncateMaxLength(fp.SanitizePath(item.SubFolder, false)),
		)
	} else {
		return user.GetUserTag()
	}
}
