package gdrive

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/DaRealFreak/watcher-go/pkg/models"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/api/drive/v3"
)

func (m *gdrive) downloadFiles(sortedFiles []*drive.File, item *models.TrackedItem) (err error) {
	log.WithField("module", m.Key).Info(
		fmt.Sprintf(`found %d new items for uri: "%s"`, len(sortedFiles), item.URI),
	)

	for index, file := range sortedFiles {
		log.WithField("module", m.Key).Info(
			fmt.Sprintf(
				`downloading updates for uri: "%s" (%0.2f%%)`,
				item.URI,
				float64(index+1)/float64(len(sortedFiles))*100,
			),
		)

		var res *http.Response

		for try := 0; try < 5; try++ {
			log.WithField("module", m.Key).Debugf(
				`downloading google drive file "%s" (try: %d)`, file.Name, try+1,
			)

			res, err = m.driveService.Files.Get(file.Id).Download()
			if err == nil {
				break
			}

			time.Sleep(time.Duration(try) * 5)
		}

		if err != nil {
			return err
		}

		localFilePath := filepath.Join(
			viper.GetString("download.directory"),
			m.Key, file.Name,
		)
		m.Session.EnsureDownloadDirectory(localFilePath)

		localFile, err := os.Create(localFilePath)
		if err != nil {
			return err
		}

		written, err := io.Copy(localFile, res.Body)
		if err != nil {
			return err
		}

		if err := m.Session.CheckDownloadedFileForErrors(written, res.Header); err != nil {
			return err
		}

		modified, _ := time.Parse(time.RFC3339, file.ModifiedTime)

		m.DbIO.UpdateTrackedItem(item, strconv.Itoa(int(modified.Unix())))
	}

	return nil
}
