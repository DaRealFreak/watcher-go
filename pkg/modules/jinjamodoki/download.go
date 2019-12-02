package jinjamodoki

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path"
	"strings"
	"time"

	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/PuerkitoBio/goquery"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/time/rate"
)

// ProcessDownloadQueue processes the default download queue, can be used if the module doesn't require special actions
func (m *jinjaModoki) processDownloadQueue(queue []models.DownloadQueueItem, item *models.TrackedItem) error {
	// only the downloads have a rate limit, so we only set it here
	m.defaultSession.RateLimiter = rate.NewLimiter(rate.Every(1*time.Second), 1)

	log.WithField("module", m.Key).Info(
		fmt.Sprintf("found %d new items for uri: \"%s\"", len(queue), item.URI),
	)

	for index, data := range queue {
		log.WithField("module", m.Key).Info(
			fmt.Sprintf(
				"downloading updates for uri: \"%s\" (%0.2f%%)",
				item.URI,
				float64(index+1)/float64(len(queue))*100,
			),
		)

		if err := m.downloadItem(data, item); err != nil {
			return err
		}
	}

	// remove rate limiter again for next item
	m.defaultSession.RateLimiter = nil

	return nil
}

func (m *jinjaModoki) downloadItem(data models.DownloadQueueItem, item *models.TrackedItem) error {
	filePath := path.Join(viper.GetString("download.directory"), m.Key, data.DownloadTag, data.FileName)

	err := m.defaultSession.DownloadFile(
		filePath,
		data.FileURI,
	)
	if err != nil {
		return err
	}

	if err := m.checkDownloadedFileForErrors(filePath); err != nil {
		if err := m.setProxyMethod(); err != nil {
			return err
		}

		return m.downloadItem(data, item)
	}

	m.DbIO.UpdateTrackedItem(item, data.ItemID)

	return nil
}

// checkDownloadedFileForErrors checks the already downloaded file for validity and possible rating limitations
// possible user limitations:
// - 100 MB/5 min
// - 500 MB/60 min
// - 1 GB/24 hours
// possible website limitations:
// - 5 GB/5 min
// - 20 GB/60 min
// - 800 GB/24 hours
// possible proxy limitations:
// - website doesn't allow public proxies, maybe list got updated
func (m *jinjaModoki) checkDownloadedFileForErrors(filePath string) error {
	// #nosec
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		// error reading file, not downloaded properly?
		return err
	}

	if !strings.Contains(string(content), "<!-- ERROR MESSAGE -->") {
		// no errors
		return nil
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(content))
	if err != nil {
		return err
	}

	return fmt.Errorf(doc.Find("p > b").First().Text())
}
