package chounyuu

import (
	"fmt"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
	log "github.com/sirupsen/logrus"
	"path"
)

func (m *chounyuu) processDownloadQueue(downloadQueue []models.DownloadQueueItem, trackedItem *models.TrackedItem, notifications ...*models.Notification) error {
	log.WithField("module", m.Key).Info(
		fmt.Sprintf("found %d new items for uri: \"%s\"", len(downloadQueue), trackedItem.URI),
	)

	if notifications != nil {
		for _, notification := range notifications {
			log.WithField("module", m.Key).Log(
				notification.Level,
				notification.Message,
			)
		}
	}

	for index, data := range downloadQueue {
		log.WithField("module", m.Key).Info(
			fmt.Sprintf(
				"downloading updates for uri: \"%s\" (%0.2f%%)",
				trackedItem.URI,
				float64(index+1)/float64(len(downloadQueue))*100,
			),
		)

		err := m.Session.DownloadFile(
			path.Join(
				m.GetDownloadDirectory(),
				m.Key,
				fp.TruncateMaxLength(fp.SanitizePath(trackedItem.SubFolder, false)),
				fp.TruncateMaxLength(fp.SanitizePath(data.DownloadTag, false)),
				fp.TruncateMaxLength(fp.SanitizePath(data.FileName, false)),
			),
			data.FileURI,
		)
		if err != nil {
			switch err.(type) {
			case DeletedMediaError:
				log.WithField("module", m.ModuleKey()).Warnf(
					fmt.Sprintf("received 404 status code for URI \"%s\", content got most likely deleted, skipping", data.FileURI),
				)
			default:
				return err
			}
		}

		m.DbIO.UpdateTrackedItem(trackedItem, data.ItemID)
	}

	return nil
}
