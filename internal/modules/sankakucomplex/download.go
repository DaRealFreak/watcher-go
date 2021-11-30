package sankakucomplex

import (
	"fmt"
	"net/url"
	"path"
	"strconv"
	"time"

	"github.com/DaRealFreak/watcher-go/internal/models"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func (m *sankakuComplex) processDownloadQueue(downloadQueue []models.DownloadQueueItem, trackedItem *models.TrackedItem) error {
	log.WithField("module", m.Key).Info(
		fmt.Sprintf("found %d new items for uri: \"%s\"", len(downloadQueue), trackedItem.URI),
	)

	for index, data := range downloadQueue {
		log.WithField("module", m.Key).Info(
			fmt.Sprintf(
				"downloading updates for uri: \"%s\" (%0.2f%%)",
				trackedItem.URI,
				float64(index+1)/float64(len(downloadQueue))*100,
			),
		)

		u, err := url.Parse(data.FileURI)
		if err != nil {
			return err
		}

		parsedQuery, _ := url.ParseQuery(u.RawQuery)
		if expires := parsedQuery.Get("expires"); expires != "" {
			var i int64
			i, err = strconv.ParseInt(expires, 10, 64)
			if err != nil {
				return err
			}

			if time.Now().Unix() >= i {
				log.WithField("module", m.Key).Info(
					fmt.Sprintf(
						"links expired for uri, refreshing progress: \"%s\"",
						trackedItem.URI,
					),
				)

				return m.Parse(trackedItem)
			}
		}

		err = m.Session.DownloadFile(
			path.Join(viper.GetString("download.directory"), m.Key, data.DownloadTag, data.FileName),
			data.FileURI,
		)
		if err != nil {
			return err
		}

		m.DbIO.UpdateTrackedItem(trackedItem, data.ItemID)
	}

	return nil
}
