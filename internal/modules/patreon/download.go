package patreon

import (
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// postDownload is the struct used for downloading post contents
type postDownload struct {
	PostID       int
	CreatorID    int
	CreatorName  string
	PatreonURL   string
	Attachments  []*campaignInclude
	ExternalURLs []string
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
				fileName := fp.SanitizePath(attachment.Attributes.Name, false)
				if err := m.Session.DownloadFile(
					path.Join(
						viper.GetString("download.directory"),
						m.Key,
						strings.TrimSpace(fmt.Sprintf("%d_%s", data.CreatorID, data.CreatorName)),
						fp.TruncateMaxLength(strings.TrimSpace(fmt.Sprintf("%d_%s", data.PostID, fileName))),
					),
					attachment.Attributes.URL,
				); err != nil {
					return err
				}
			default:
				// if no download URL is returned from the API we don't have the reward unlocked and can't download it
				if attachment.Attributes.DownloadURL == "" {
					log.WithField("module", m.Key).Warningf(
						"post %s not unlocked, skipping attachment %s",
						data.PatreonURL,
						attachment.ID,
					)

					continue
				}

				fileName := fp.SanitizePath(fp.GetFileName(attachment.Attributes.FileName), false)
				if err := m.Session.DownloadFile(
					path.Join(
						viper.GetString("download.directory"),
						m.Key,
						strings.TrimSpace(fmt.Sprintf("%d_%s", data.CreatorID, data.CreatorName)),
						fp.TruncateMaxLength(strings.TrimSpace(fmt.Sprintf("%d_%s", data.PostID, fileName))),
					),
					attachment.Attributes.DownloadURL,
				); err != nil {
					return err
				}
			}
		}

		for _, externalURL := range data.ExternalURLs {
			module := modules.GetModuleFactory().GetModuleFromURI(externalURL)
			module.InitializeModule()
			newItem := m.DbIO.GetFirstOrCreateTrackedItem(externalURL, "", module)
			if m.Cfg.Run.ForceNew && newItem.CurrentItem != "" {
				log.WithField("module", m.Key).Info(
					fmt.Sprintf("resetting progress for item %s (current id: %s)", newItem.URI, newItem.CurrentItem),
				)
				newItem.CurrentItem = ""
				m.DbIO.ChangeTrackedItemCompleteStatus(newItem, false)
				m.DbIO.UpdateTrackedItem(newItem, "")
			}

			if err := module.Parse(newItem); err != nil {
				return err
			}
		}

		m.DbIO.UpdateTrackedItem(item, strconv.Itoa(data.PostID))
	}

	return nil
}
