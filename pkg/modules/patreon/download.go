package patreon

import (
	"fmt"
	"path"
	"strconv"

	"github.com/DaRealFreak/watcher-go/pkg/models"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// postDownload is the struct used for downloading post contents
type postDownload struct {
	PostID      int
	CreatorID   int
	CreatorName string
	Attachments []*campaignInclude
}

func (m *patreon) processDownloadQueue(downloadQueue []*postDownload, item *models.TrackedItem) error {
	log.WithField("module", m.Key).Info(
		fmt.Sprintf("found %d new items for uri: \"%s\"", len(downloadQueue), item.URI),
	)

	for index, data := range downloadQueue {
		log.WithField("module", m.Key).Info(
			fmt.Sprintf(
				"downloading updates for uri: \"%s\" (%0.2f%%)",
				item.URI,
				float64(index+1)/float64(len(downloadQueue))*100,
			),
		)

		for _, attachment := range data.Attachments {
			switch attachment.Type {
			case "attachment":
				if err := m.Session.DownloadFile(
					path.Join(
						viper.GetString("download.directory"),
						m.Key,
						fmt.Sprintf("%d_%s", data.CreatorID, data.CreatorName),
						fmt.Sprintf("%d_%s", data.PostID, attachment.Attributes.Name),
					),
					attachment.Attributes.URL,
				); err != nil {
					return err
				}
			default:
				if err := m.Session.DownloadFile(
					path.Join(
						viper.GetString("download.directory"),
						m.Key,
						fmt.Sprintf("%d_%s", data.CreatorID, data.CreatorName),
						fmt.Sprintf("%d_%s", data.PostID, attachment.Attributes.FileName),
					),
					attachment.Attributes.DownloadURL,
				); err != nil {
					return err
				}
			}
		}

		m.DbIO.UpdateTrackedItem(item, strconv.Itoa(data.PostID))
	}

	return nil
}
