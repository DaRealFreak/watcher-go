package twitter

import (
	"fmt"
	"path"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules/twitter/graphql_api"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
	"log/slog"
)

func (m *twitter) processDownloadQueueGraphQL(downloadQueue []*graphql_api.Tweet, trackedItem *models.TrackedItem) error {
	slog.Info(fmt.Sprintf("found %d new items for uri: \"%s\"", len(downloadQueue), trackedItem.URI), "module", m.Key)

	for index, tweet := range downloadQueue {
		slog.Info(fmt.Sprintf(
				"downloading updates for uri: \"%s\" (%0.2f%%)",
				trackedItem.URI,
				float64(index+1)/float64(len(downloadQueue))*100,
			), "module", m.Key)

		downloadItems := tweet.DownloadItems()
		for i := range downloadItems {
			// iterate reverse over the download items to download the cover image last
			downloadItem := downloadItems[i]
			err := m.twitterGraphQlAPI.Session.DownloadFile(
				path.Join(
					m.GetDownloadDirectory(),
					m.Key,
					fp.TruncateMaxLength(fp.SanitizePath(m.getDownloadTag(trackedItem, downloadItem), false)),
					fp.TruncateMaxLength(fp.SanitizePath(downloadItem.FileName, false)),
				),
				downloadItem.FileURI,
			)
			if err != nil {
				switch err.(type) {
				case graphql_api.DMCAError:
					slog.Warn(fmt.Sprintf("received 403 status code for URI \"%s\", content got most likely DMCA'd, skipping",
						downloadItem.FileURI,), "module", m.ModuleKey())
				case graphql_api.DeletedMediaError:
					slog.Warn(fmt.Sprintf("received 404 status code for URI \"%s\", content got most likely deleted, skipping",
						downloadItem.FileURI,), "module", m.ModuleKey())
				default:
					return err
				}
			}
		}
		m.DbIO.UpdateTrackedItem(trackedItem, tweet.Item.ItemContent.TweetResults.Result.TweetData().RestID.String())
	}

	return nil
}

func (m *twitter) getDownloadTag(item *models.TrackedItem, downloadItem *models.DownloadQueueItem) string {
	if m.settings.UseSubFolderForAuthorName && item.SubFolder != "" {
		return item.SubFolder
	}

	return downloadItem.DownloadTag
}
