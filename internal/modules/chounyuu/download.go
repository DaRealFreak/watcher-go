package chounyuu

import (
	"fmt"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
	"log/slog"
	"path"
	"context"
)

func (m *chounyuu) processDownloadQueue(downloadQueue []models.DownloadQueueItem, trackedItem *models.TrackedItem, notifications ...*models.Notification) error {
	slog.Info(fmt.Sprintf("found %d new items for uri: \"%s\"", len(downloadQueue), trackedItem.URI), "module", m.Key)

	for _, notification := range notifications {
		slog.Log(context.Background(), 
			notification.Level, notification.Message, "module", m.Key)
	}

	for index, data := range downloadQueue {
		slog.Info(fmt.Sprintf(
				"downloading updates for uri: \"%s\" (%0.2f%%)",
				trackedItem.URI,
				float64(index+1)/float64(len(downloadQueue))*100,
			), "module", m.Key)

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
				slog.Warn(fmt.Sprintf("received 404 status code for URI \"%s\", content got most likely deleted, skipping",
					data.FileURI,), "module", m.ModuleKey())
			default:
				return err
			}
		}

		m.DbIO.UpdateTrackedItem(trackedItem, data.ItemID)
	}

	return nil
}
