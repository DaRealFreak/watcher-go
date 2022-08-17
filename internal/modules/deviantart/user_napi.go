// Package deviantart contains the implementation of the deviantart module
// nolint: dupl
package deviantart

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules/deviantart/napi"
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

	response, err := m.nAPI.DeviationsUser(username, 0, 0, napi.MaxLimit, true)
	if err != nil {
		return err
	}

	for !foundCurrentItem {
		for _, result := range response.Deviations {
			t, dateErr := time.Parse(napi.DateLayout, result.Deviation.PublishedTime)
			if dateErr != nil {
				return dateErr
			}

			if item.CurrentItem == "" || t.Unix() > currentItemID {
				downloadQueue = append(downloadQueue, downloadQueueItemNAPI{
					itemID:      strconv.Itoa(int(t.Unix())),
					deviation:   result.Deviation,
					downloadTag: m.SanitizePath(result.Deviation.Author.Username, false),
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
