package pixiv

import (
	"strconv"

	"github.com/DaRealFreak/watcher-go/pkg/models"
	pixivapi "github.com/DaRealFreak/watcher-go/pkg/modules/pixiv/pixiv_api"
	log "github.com/sirupsen/logrus"
)

func (m *pixiv) parseUser(item *models.TrackedItem) error {
	userID, _ := strconv.ParseInt(m.patterns.memberPattern.FindStringSubmatch(item.URI)[1], 10, 64)

	_, err := m.mobileAPI.GetUserDetail(int(userID))
	if err != nil {
		switch err.(type) {
		case pixivapi.UserUnavailableError:
			log.WithField("module", m.Key).Warning(
				"couldn't retrieve user details, changing artist to complete",
			)
			m.DbIO.ChangeTrackedItemCompleteStatus(item, true)

			return nil
		default:
			return err
		}
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
					DownloadTag:  illustration.User.GetUserTag(),
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
