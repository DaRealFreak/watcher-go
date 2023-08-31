package deviantart

import (
	"fmt"
	"net/url"
	"path"
	"strconv"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules/deviantart/api"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
)

func (m *deviantArt) parseCollectionDevAPI(item *models.TrackedItem) error {
	username := m.daPattern.collectionPattern.FindStringSubmatch(item.URI)[1]
	collectionID := m.daPattern.collectionPattern.FindStringSubmatch(item.URI)[2]
	collectionIntID, _ := strconv.ParseInt(collectionID, 10, 64)

	collectionName, err := m.daAPI.CollectionNameFromURL(item.URI)
	if err != nil {
		return err
	} else if collectionName == "All" {
		// special case since the API won't return the "All" folder, and we can't retrieve the folder ID
		// with the Eclipse theme which is enforced for new users....
		return m.parseAllCollectionsDevAPI(username)
	}

	collectionUUID, err := m.daAPI.CollectionFolderIDToUUID(username, int(collectionIntID))
	if err != nil {
		return err
	}

	downloadQueue, err := m.getCollectionDownloadQueueDevAPI(item, username, collectionUUID)
	if err != nil {
		return err
	}

	return m.processDownloadQueue(downloadQueue, item)
}

// parseAllCollections parses all favourites of the passed user
func (m *deviantArt) parseAllCollectionsDevAPI(user string) error {
	folderResults, err := m.daAPI.CollectionFolders(user, 0, api.MaxDeviationsPerPage)
	if err != nil {
		return err
	}

	for _, folder := range folderResults.Results {
		appURL := fmt.Sprintf("deviantart://collection/%s/%s", user, url.PathEscape(folder.FolderUUID))
		m.DbIO.GetFirstOrCreateTrackedItem(appURL, "", m)
	}

	for folderResults.NextOffset != nil && folderResults.HasMore {
		folderResults, err = m.daAPI.CollectionFolders(user, uint(*folderResults.NextOffset), api.MaxDeviationsPerPage)
		if err != nil {
			return err
		}

		for _, folder := range folderResults.Results {
			appURL := fmt.Sprintf("deviantart://collection/%s/%s", user, url.PathEscape(folder.FolderUUID))
			m.DbIO.GetFirstOrCreateTrackedItem(appURL, "", m)
		}
	}

	return nil
}

func (m *deviantArt) parseCollectionUUIDDevAPI(item *models.TrackedItem) error {
	username := m.daPattern.collectionUUIDPattern.FindStringSubmatch(item.URI)[1]
	collectionUUID := m.daPattern.collectionUUIDPattern.FindStringSubmatch(item.URI)[2]

	downloadQueue, err := m.getCollectionDownloadQueueDevAPI(item, username, collectionUUID)
	if err != nil {
		return err
	}

	return m.processDownloadQueue(downloadQueue, item)
}

func (m *deviantArt) getCollectionDownloadQueueDevAPI(
	item *models.TrackedItem, username string, collectionUUID string,
) ([]downloadQueueItemDevAPI, error) {
	var downloadQueue []downloadQueueItemDevAPI

	currentItemID, _ := strconv.ParseInt(item.CurrentItem, 10, 64)
	foundCurrentItem := false

	collectionName, err := m.daAPI.CollectionNameFromUUID(username, collectionUUID)
	if err != nil {
		return nil, err
	}

	if item.SubFolder == "" {
		m.DbIO.ChangeTrackedItemSubFolder(item, path.Join(
			fp.SanitizePath(username, false),
			fp.SanitizePath(collectionName, false),
		))
	}

	response, err := m.daAPI.Collection(username, collectionUUID, 0, api.MaxDeviationsPerPage)
	if err != nil {
		return nil, err
	}

	for !foundCurrentItem {
		for _, deviation := range response.Results {
			var publishedTime int64
			publishedTime, err = strconv.ParseInt(deviation.PublishedTime, 10, 64)
			if err != nil {
				return nil, err
			}

			if item.CurrentItem == "" || publishedTime > currentItemID {
				downloadQueue = append(downloadQueue, downloadQueueItemDevAPI{
					itemID:      deviation.PublishedTime,
					deviation:   deviation,
					downloadTag: fp.SanitizePath(item.SubFolder, true),
				})
			} else {
				foundCurrentItem = true
				break
			}
		}

		if response.NextOffset == nil || foundCurrentItem {
			break
		}

		response, err = m.daAPI.Collection(username, collectionUUID, *response.NextOffset, api.MaxDeviationsPerPage)
		if err != nil {
			return nil, err
		}
	}

	for i, j := 0, len(downloadQueue)-1; i < j; i, j = i+1, j-1 {
		downloadQueue[i], downloadQueue[j] = downloadQueue[j], downloadQueue[i]
	}

	return downloadQueue, nil
}
