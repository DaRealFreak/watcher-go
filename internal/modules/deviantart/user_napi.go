package deviantart

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules/deviantart/napi"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
	log "github.com/sirupsen/logrus"
)

func (m *deviantArt) parseUserNapi(item *models.TrackedItem) error {
	var downloadQueue []downloadQueueItemNAPI

	username := m.daPattern.userPattern.FindStringSubmatch(item.URI)[1]
	currentItemID, _ := strconv.ParseInt(item.CurrentItem, 10, 64)
	foundCurrentItem := false

	userInfo, err := m.nAPI.UserInfo(username, napi.UserInfoExpandDefault)
	if err != nil {
		return err
	}

	if strings.ToLower(userInfo.User.Username) != username {
		uri := fmt.Sprintf("https://www.deviantart.com/%s", userInfo.User.GetUsernameUrl())
		log.WithField("module", m.ModuleKey()).Warnf(
			"author changed its name, updated tracked uri from \"%s\" to \"%s\"",
			item.URI,
			uri,
		)

		m.DbIO.ChangeTrackedItemUri(item, uri)
	}

	if item.SubFolder == "" {
		m.DbIO.ChangeTrackedItemSubFolder(item, userInfo.User.Username)
	}

	response, err := m.nAPI.DeviationsUser(username, 0, 0, napi.MaxLimit, true)
	if err != nil {
		return err
	}

	for !foundCurrentItem {
		for _, result := range response.Deviations {
			if item.CurrentItem == "" || result.Deviation.GetPublishedTime().Unix() > currentItemID {
				downloadQueue = append(downloadQueue, downloadQueueItemNAPI{
					itemID:      result.Deviation.GetPublishedTimestamp(),
					deviation:   result.Deviation,
					downloadTag: fp.SanitizePath(item.SubFolder, false),
				})
			} else {
				foundCurrentItem = true
				break
			}
		}

		if response.NextOffset == nil || foundCurrentItem {
			break
		}

		nextOffset, _ := response.NextOffset.Int64()
		response, err = m.nAPI.DeviationsUser(username, 0, int(nextOffset), napi.MaxLimit, true)
		if err != nil {
			return err
		}
	}

	for i, j := 0, len(downloadQueue)-1; i < j; i, j = i+1, j-1 {
		downloadQueue[i], downloadQueue[j] = downloadQueue[j], downloadQueue[i]
	}

	return m.processDownloadQueueNapi(downloadQueue, item)
}
