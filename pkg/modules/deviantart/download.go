package deviantart

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/DaRealFreak/watcher-go/pkg/imaging/duplication"
	watcherIO "github.com/DaRealFreak/watcher-go/pkg/io"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/DaRealFreak/watcher-go/pkg/modules/deviantart/api"
	"github.com/jaytaylor/html2text"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type downloadQueueItem struct {
	itemID      string
	deviation   *api.Deviation
	downloadTag string
}

type downloadLog struct {
	download string
	content  string
}

func (l *downloadLog) downloadedFiles() (downloadedFiles []string) {
	for _, item := range []string{l.download, l.content} {
		if item != "" {
			downloadedFiles = append(downloadedFiles, item)
		}
	}

	return downloadedFiles
}

func (m *deviantArt) processDownloadQueue(downloadQueue []downloadQueueItem, trackedItem *models.TrackedItem) error {
	log.WithField("module", m.Key).Info(
		fmt.Sprintf("found %d new items for uri: %s", len(downloadQueue), trackedItem.URI),
	)

	for index, deviationItem := range downloadQueue {
		log.WithField("module", m.Key).Info(
			fmt.Sprintf(
				"downloading updates for uri: %s (%0.2f%%)",
				trackedItem.URI,
				float64(index+1)/float64(len(downloadQueue))*100,
			),
		)

		// ensure download directory, needed for only text artists
		m.daAPI.Session.EnsureDownloadDirectory(
			path.Join(
				viper.GetString("download.directory"),
				m.Key,
				deviationItem.deviation.Author.Username,
				"tmp.txt",
			),
		)

		var downloadLog downloadLog

		if deviationItem.deviation.Excerpt != nil {
			if err := m.downloadHTMLContent(deviationItem.deviation); err != nil {
				return err
			}
		}

		if deviationItem.deviation.IsDownloadable {
			if err := m.downloadDeviation(deviationItem.deviation, &downloadLog); err != nil {
				return err
			}
		}

		if err := m.downloadContent(deviationItem.deviation, &downloadLog); err != nil {
			return err
		}

		m.DbIO.UpdateTrackedItem(trackedItem, deviationItem.deviation.PublishedTime)
	}

	return nil
}

func (m *deviantArt) downloadContent(deviation *api.Deviation, downloadLog *downloadLog) error {
	if deviation.DeviationDownload != nil && deviation.Content != nil {
		tmpFile, err := ioutil.TempFile("", ".*")
		if err != nil {
			return err
		}

		if err := m.daAPI.Session.DownloadFile(tmpFile.Name(), deviation.Content.Src); err != nil {
			return err
		}

		sim, err := duplication.CheckForSimilarity(downloadLog.download, tmpFile.Name())
		// if either the file couldn't be converted (probably different file type) or similarity is below 95%
		if err != nil && sim <= 0.95 {
			downloadLog.content, _ = filepath.Abs(path.Join(viper.GetString("download.directory"),
				m.Key,
				deviation.Author.Username,
				fmt.Sprintf(
					"%s_c_%s%s",
					deviation.PublishedTime,
					strings.ReplaceAll(m.SanitizePath(deviation.Title, false), " ", "_"),
					m.GetFileExtension(deviation.Content.Src),
				),
			))
			if err := watcherIO.CopyFile(tmpFile.Name(), downloadLog.content); err != nil {
				return err
			}
		}
	} else if deviation.Content != nil {
		downloadLog.content, _ = filepath.Abs(path.Join(viper.GetString("download.directory"),
			m.Key,
			deviation.Author.Username,
			fmt.Sprintf(
				"%s_c_%s%s",
				deviation.PublishedTime,
				strings.ReplaceAll(m.SanitizePath(deviation.Title, false), " ", "_"),
				m.GetFileExtension(deviation.Content.Src),
			),
		))
		if err := m.daAPI.Session.DownloadFile(downloadLog.content, deviation.Content.Src); err != nil {
			return err
		}
	}

	return m.downloadThumbs(deviation, downloadLog)
}

func (m *deviantArt) downloadThumbs(deviation *api.Deviation, downloadLog *downloadLog) error {
	// compare thumb and download/content
	if len(deviation.Thumbs) == 0 {
		return nil
	}

	lastThumb := deviation.Thumbs[len(deviation.Thumbs)-1]

	tmpFile, err := ioutil.TempFile("", ".*")
	if err != nil {
		return err
	}

	if err := m.daAPI.Session.DownloadFile(tmpFile.Name(), lastThumb.Src); err != nil {
		return err
	}

	if len(downloadLog.downloadedFiles()) == 0 {
		if err := watcherIO.CopyFile(tmpFile.Name(), path.Join(viper.GetString("download.directory"),
			m.Key,
			deviation.Author.Username,
			fmt.Sprintf(
				"%s_tmb_%s%s",
				deviation.PublishedTime,
				strings.ReplaceAll(m.SanitizePath(deviation.Title, false), " ", "_"),
				m.GetFileExtension(lastThumb.Src),
			),
		)); err != nil {
			return err
		}

		return nil
	}

	for _, item := range downloadLog.downloadedFiles() {
		sim, _ := duplication.CheckForSimilarity(item, tmpFile.Name())
		// if either the file couldn't be converted (probably different file type) or similarity is below 95%
		if sim > 0.95 {
			log.WithField("module", m.Key).Debugf(
				"thumbnail matching with %s above threshold %.2f%%", item, sim*100,
			)

			return nil
		}
	}

	return watcherIO.CopyFile(tmpFile.Name(), path.Join(viper.GetString("download.directory"),
		m.Key,
		deviation.Author.Username,
		fmt.Sprintf(
			"%s_tmb_%s%s",
			deviation.PublishedTime,
			strings.ReplaceAll(m.SanitizePath(deviation.Title, false), " ", "_"),
			m.GetFileExtension(lastThumb.Src),
		)),
	)
}

func (m *deviantArt) downloadDeviation(deviation *api.Deviation, downloadLog *downloadLog) error {
	deviationDownload, err := m.daAPI.DeviationDownloadFallback(deviation.DeviationURL)
	if err != nil {
		deviationDownload, err = m.daAPI.DeviationDownload(deviation.DeviationID)
		if err != nil {
			return err
		}
	}

	deviation.DeviationDownload = deviationDownload
	downloadLog.download, _ = filepath.Abs(path.Join(viper.GetString("download.directory"),
		m.Key,
		deviation.Author.Username,
		fmt.Sprintf(
			"%s_d_%s%s",
			deviation.PublishedTime,
			strings.ReplaceAll(m.SanitizePath(deviation.Title, false), " ", "_"),
			m.GetFileExtension(deviationDownload.Src),
		),
	))

	if err := m.daAPI.Session.DownloadFile(
		downloadLog.download,
		deviationDownload.Src,
	); err != nil {
		return err
	}

	return nil
}

func (m *deviantArt) downloadHTMLContent(deviation *api.Deviation) error {
	// deviation has text so we retrieve the full content
	deviationContent, err := m.daAPI.DeviationContent(deviation.DeviationID)
	if err != nil {
		return err
	}

	text, err := html2text.FromString(deviationContent.HTML)
	if err != nil {
		return err
	}

	filePath := path.Join(viper.GetString("download.directory"),
		m.Key,
		deviation.Author.Username,
		fmt.Sprintf(
			"%s_t_%s.txt",
			deviation.PublishedTime,
			strings.ReplaceAll(m.SanitizePath(deviation.Title, false), " ", "_"),
		),
	)
	if err := ioutil.WriteFile(filePath, []byte(text), os.ModePerm); err != nil {
		return err
	}

	return nil
}
