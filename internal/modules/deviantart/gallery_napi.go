package deviantart

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules/deviantart/napi"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
	log "github.com/sirupsen/logrus"
)

func (m *deviantArt) parseGalleryNapi(item *models.TrackedItem) error {
	username := m.daPattern.galleryPattern.FindStringSubmatch(item.URI)[1]
	galleryID := m.daPattern.galleryPattern.FindStringSubmatch(item.URI)[2]
	galleryIntID, _ := strconv.ParseInt(galleryID, 10, 64)

	galleries, err := m.nAPI.GalleriesOverviewUser(username, napi.MaxLimit, true)
	if err != nil {
		return err
	}

	galleryFolder := galleries.FindFolderByFolderId(int(galleryIntID))
	if galleryFolder == nil {
		return fmt.Errorf("unable to find gallery")
	}

	if strings.ToLower(galleryFolder.Owner.Username) != username {
		uri := fmt.Sprintf("https://www.deviantart.com/%s/gallery/%d", strings.ToLower(galleryFolder.Owner.Username), galleryIntID)
		log.WithField("module", m.ModuleKey()).Warnf(
			"author changed its name, updated tracked uri from \"%s\" to \"%s\"",
			item.URI,
			uri,
		)

		m.DbIO.ChangeTrackedItemUri(item, uri)
	}

	return m.parseGalleryByFolderNapi(item, galleryFolder)
}

func (m *deviantArt) parseGalleryByFolderNapi(item *models.TrackedItem, galleryFolder *napi.Folder) error {
	var downloadQueue []downloadQueueItemNAPI

	foundCurrentItem := false

	galleryId, _ := galleryFolder.FolderId.Int64()
	response, err := m.nAPI.DeviationsUser(galleryFolder.Owner.Username, int(galleryId), 0, napi.MaxLimit, false)
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
						fp.SanitizePath(galleryFolder.Owner.Username, false),
						fmt.Sprintf(
							"%s_%s",
							galleryFolder.FolderId.String(),
							fp.SanitizePath(galleryFolder.Name, false),
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
		response, err = m.nAPI.DeviationsUser(galleryFolder.Owner.Username, int(galleryId), int(nextOffSet), napi.MaxLimit, false)
		if err != nil {
			return err
		}
	}

	for i, j := 0, len(downloadQueue)-1; i < j; i, j = i+1, j-1 {
		downloadQueue[i], downloadQueue[j] = downloadQueue[j], downloadQueue[i]
	}

	return m.processDownloadQueueNapi(downloadQueue, item)
}
