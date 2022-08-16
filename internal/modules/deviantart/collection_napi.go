package deviantart

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/DaRealFreak/watcher-go/pkg/fp"

	"github.com/DaRealFreak/watcher-go/internal/modules/deviantart/napi"

	"github.com/DaRealFreak/watcher-go/internal/models"
)

func (m *deviantArt) parseCollectionUUIDNapi(item *models.TrackedItem) error {
	username := m.daPattern.collectionUUIDPattern.FindStringSubmatch(item.URI)[1]
	collectionUUID := m.daPattern.collectionUUIDPattern.FindStringSubmatch(item.URI)[2]

	favoritesOverView, err := m.nAPI.FavoritesOverviewUser(username, napi.MaxLimit, true)
	if err != nil {
		return err
	}

	collectionFolder := favoritesOverView.FindFolderByFolderUuid(collectionUUID)
	if collectionFolder == nil {
		return fmt.Errorf("unable to find collection")
	}

	return m.parseCollectionByFolderNapi(item, collectionFolder)
}

func (m *deviantArt) parseCollectionNapi(item *models.TrackedItem) error {
	username := m.daPattern.collectionPattern.FindStringSubmatch(item.URI)[1]
	collectionID := m.daPattern.collectionPattern.FindStringSubmatch(item.URI)[2]
	collectionIntID, _ := strconv.ParseInt(collectionID, 10, 64)

	favoritesOverView, err := m.nAPI.FavoritesOverviewUser(username, napi.MaxLimit, true)
	if err != nil {
		return err
	}

	collectionFolder := favoritesOverView.FindFolderByFolderId(int(collectionIntID))
	if collectionFolder == nil {
		return fmt.Errorf("unable to find collection")
	}

	return m.parseCollectionByFolderNapi(item, collectionFolder)
}

func (m *deviantArt) parseCollectionByFolderNapi(item *models.TrackedItem, collectionFolder *napi.Folder) error {
	var downloadQueue []downloadQueueItemNAPI

	foundCurrentItem := false

	collectionId, _ := collectionFolder.FolderId.Int64()
	response, err := m.nAPI.FavoritesUser(collectionFolder.Owner.Username, int(collectionId), 0, napi.MaxLimit, false)
	if err != nil {
		return err
	}

	for !foundCurrentItem {
		for _, deviation := range response.Deviations {
			if deviation.Deviation.Type == "tier" {
				// tier entries do not respect the "most-recent" order and have no content most of the time
				continue
			}

			if item.CurrentItem != deviation.Deviation.DeviationId.String() {
				downloadQueue = append(downloadQueue, downloadQueueItemNAPI{
					itemID:    deviation.Deviation.DeviationId.String(),
					deviation: deviation.Deviation,
					downloadTag: filepath.Join(
						fp.SanitizePath(collectionFolder.Owner.Username, false),
						fmt.Sprintf(
							"%s_%s",
							collectionFolder.FolderId.String(),
							fp.SanitizePath(collectionFolder.Name, false),
						),
					),
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
	}

	for i, j := 0, len(downloadQueue)-1; i < j; i, j = i+1, j-1 {
		downloadQueue[i], downloadQueue[j] = downloadQueue[j], downloadQueue[i]
	}

	return m.processDownloadQueueNapi(downloadQueue, item)
}
