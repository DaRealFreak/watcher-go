package deviantart

import (
	"fmt"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules/deviantart/napi"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
	log "github.com/sirupsen/logrus"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
)

func (m *deviantArt) parseGalleryNapi(item *models.TrackedItem) error {
	username := m.daPattern.galleryPattern.FindStringSubmatch(item.URI)[1]
	galleryID := m.daPattern.galleryPattern.FindStringSubmatch(item.URI)[2]
	galleryIntID, _ := strconv.ParseInt(galleryID, 10, 64)

	res, err := m.nAPI.CollectionsUser(username, 0, napi.CollectionLimit, napi.FolderTypeGallery, false, true)
	if err != nil {
		return err
	}

	if m.settings.MultiProxy {
		raven.CheckError(m.setProxyMethod())
	}

	galleryFolder := res.FindFolderByFolderId(int(galleryIntID))
	for galleryFolder == nil && res.HasMore {
		nextOffset, _ := res.NextOffset.Int64()
		res, err = m.nAPI.CollectionsUser(username, int(nextOffset), napi.CollectionLimit, napi.FolderTypeGallery, false, false)
		if err != nil {
			return err
		}

		if m.settings.MultiProxy {
			raven.CheckError(m.setProxyMethod())
		}

		galleryFolder = res.FindFolderByFolderId(int(galleryIntID))
	}

	if galleryFolder == nil {
		return fmt.Errorf("unable to find gallery")
	}

	if strings.ToLower(galleryFolder.Owner.Username) != username {
		uri := fmt.Sprintf(
			"https://www.deviantart.com/%s/gallery/%d/%s",
			galleryFolder.Owner.GetUsernameUrl(),
			galleryIntID,
			strings.ToLower(url.PathEscape(strings.ReplaceAll(galleryFolder.Name, " ", "-"))),
		)
		log.WithField("module", m.ModuleKey()).Warnf(
			"gallery owner changed its name, updated tracked uri from \"%s\" to \"%s\"",
			item.URI,
			uri,
		)

		m.DbIO.ChangeTrackedItemUri(item, uri)
	}

	return m.parseGalleryByFolderNapi(item, galleryFolder)
}

func (m *deviantArt) parseGalleryByFolderNapi(item *models.TrackedItem, galleryFolder *napi.Collection) error {
	var downloadQueue []downloadQueueItemNAPI

	foundCurrentItem := false

	galleryId, _ := galleryFolder.FolderId.Int64()
	response, err := m.nAPI.DeviationsUser(galleryFolder.Owner.Username, int(galleryId), 0, napi.MaxLimit, false)
	if err != nil {
		return err
	}

	if m.settings.MultiProxy {
		raven.CheckError(m.setProxyMethod())
	}

	if item.SubFolder == "" {
		m.DbIO.ChangeTrackedItemSubFolder(item, filepath.Join(
			fp.SanitizePath(galleryFolder.Owner.Username, false),
			fmt.Sprintf(
				"%s_%s",
				galleryFolder.FolderId.String(),
				fp.SanitizePath(galleryFolder.Name, false),
			),
		))
	}

	for !foundCurrentItem {
		for _, deviation := range response.Deviations {
			if deviation.Deviation.Type == "tier" {
				// tier entries do not respect the "most-recent" order and have no content most of the time
				continue
			}

			if item.CurrentItem != deviation.Deviation.DeviationId.String() {
				downloadQueue = append(downloadQueue, downloadQueueItemNAPI{
					itemID:      deviation.Deviation.DeviationId.String(),
					deviation:   deviation.Deviation,
					downloadTag: fp.SanitizePath(item.SubFolder, true),
				})
			} else {
				foundCurrentItem = true
				break
			}
		}

		if !response.HasMore || foundCurrentItem {
			break
		}

		nextOffSet, _ := response.NextOffset.Int64()
		response, err = m.nAPI.DeviationsUser(galleryFolder.Owner.Username, int(galleryId), int(nextOffSet), napi.MaxLimit, false)
		if err != nil {
			return err
		}

		if m.settings.MultiProxy {
			raven.CheckError(m.setProxyMethod())
		}
	}

	for i, j := 0, len(downloadQueue)-1; i < j; i, j = i+1, j-1 {
		downloadQueue[i], downloadQueue[j] = downloadQueue[j], downloadQueue[i]
	}

	return m.processDownloadQueueNapi(downloadQueue, item)
}
