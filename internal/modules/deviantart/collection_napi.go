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

func (m *deviantArt) parseCollectionUUIDNapi(item *models.TrackedItem) error {
	username := m.daPattern.collectionUUIDPattern.FindStringSubmatch(item.URI)[1]
	collectionUUID := m.daPattern.collectionUUIDPattern.FindStringSubmatch(item.URI)[2]

	res, err := m.nAPI.CollectionsUser(username, 0, napi.CollectionLimit, napi.FolderTypeFavourites, false, true)
	if err != nil {
		return err
	}

	if m.settings.MultiProxy {
		raven.CheckError(m.setProxyMethod())
	}

	collectionFolder := res.FindFolderByFolderUuid(collectionUUID)
	for collectionFolder == nil && res.HasMore {
		nextOffset, _ := res.NextOffset.Int64()
		res, err = m.nAPI.CollectionsUser(username, int(nextOffset), napi.CollectionLimit, napi.FolderTypeFavourites, false, false)
		if err != nil {
			return err
		}

		if m.settings.MultiProxy {
			raven.CheckError(m.setProxyMethod())
		}

		collectionFolder = res.FindFolderByFolderUuid(collectionUUID)
	}

	if collectionFolder == nil {
		return fmt.Errorf("unable to find collection")
	}

	return m.parseCollectionByFolderNapi(item, collectionFolder)
}

func (m *deviantArt) parseCollectionNapi(item *models.TrackedItem) error {
	username := m.daPattern.collectionPattern.FindStringSubmatch(item.URI)[1]
	collectionID := m.daPattern.collectionPattern.FindStringSubmatch(item.URI)[2]
	collectionIntID, _ := strconv.ParseInt(collectionID, 10, 64)

	res, err := m.nAPI.CollectionsUser(username, 0, napi.CollectionLimit, napi.FolderTypeFavourites, false, true)
	if err != nil {
		return err
	}

	if m.settings.MultiProxy {
		raven.CheckError(m.setProxyMethod())
	}

	collectionFolder := res.FindFolderByFolderId(int(collectionIntID))
	for collectionFolder == nil && res.HasMore {
		nextOffset, _ := res.NextOffset.Int64()
		res, err = m.nAPI.CollectionsUser(username, int(nextOffset), napi.CollectionLimit, napi.FolderTypeFavourites, false, false)
		if err != nil {
			return err
		}

		if m.settings.MultiProxy {
			raven.CheckError(m.setProxyMethod())
		}

		collectionFolder = res.FindFolderByFolderId(int(collectionIntID))
	}

	if collectionFolder == nil {
		return fmt.Errorf("unable to find collection")
	}

	if strings.ToLower(collectionFolder.Owner.Username) != username {
		uri := fmt.Sprintf(
			"https://www.deviantart.com/%s/favourites/%d/%s",
			collectionFolder.Owner.GetUsernameUrl(),
			collectionIntID,
			strings.ToLower(url.PathEscape(strings.ReplaceAll(collectionFolder.Name, " ", "-"))),
		)
		log.WithField("module", m.ModuleKey()).Warnf(
			"collection owner changed its name, updated tracked uri from \"%s\" to \"%s\"",
			item.URI,
			uri,
		)

		m.DbIO.ChangeTrackedItemUri(item, uri)
	}

	return m.parseCollectionByFolderNapi(item, collectionFolder)
}

func (m *deviantArt) parseCollectionByFolderNapi(item *models.TrackedItem, collectionFolder *napi.Collection) error {
	var downloadQueue []downloadQueueItemNAPI

	foundCurrentItem := false

	collectionId, _ := collectionFolder.FolderId.Int64()
	response, err := m.nAPI.FavoritesUser(collectionFolder.Owner.Username, int(collectionId), 0, napi.MaxLimit, false)
	if err != nil {
		return err
	}

	if m.settings.MultiProxy {
		raven.CheckError(m.setProxyMethod())
	}

	if item.SubFolder == "" {
		m.DbIO.ChangeTrackedItemSubFolder(item, filepath.Join(
			fp.SanitizePath(collectionFolder.Owner.Username, false),
			fmt.Sprintf(
				"%s_%s",
				collectionFolder.FolderId.String(),
				fp.SanitizePath(collectionFolder.Name, false),
			),
		))
	}

	for {
		for _, deviation := range response.Deviations {
			if deviation.Type == "tier" {
				// tier entries do not respect the "most-recent" order and have no content most of the time
				continue
			}

			if item.CurrentItem != deviation.DeviationId.String() {
				downloadQueue = append(downloadQueue, downloadQueueItemNAPI{
					itemID:      deviation.DeviationId.String(),
					deviation:   deviation,
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
		response, err = m.nAPI.FavoritesUser(collectionFolder.Owner.Username, int(collectionId), int(nextOffSet), napi.MaxLimit, false)
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
