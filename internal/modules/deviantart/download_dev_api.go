package deviantart

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules/deviantart/api"
	"github.com/DaRealFreak/watcher-go/internal/raven"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
	"github.com/DaRealFreak/watcher-go/pkg/imaging/duplication"
	watcherIO "github.com/DaRealFreak/watcher-go/pkg/io"
	"github.com/jaytaylor/html2text"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type downloadQueueItemDevAPI struct {
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

func (m *deviantArt) processDownloadQueue(downloadQueue []downloadQueueItemDevAPI, trackedItem *models.TrackedItem) error {
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
		m.daAPI.UserSession.EnsureDownloadDirectory(
			path.Join(
				viper.GetString("download.directory"),
				m.Key,
				deviationItem.downloadTag,
				"tmp.txt",
			),
		)

		var itemDownloadLog downloadLog

		if deviationItem.deviation.Excerpt != nil {
			if err := m.downloadHTMLContent(deviationItem); err != nil {
				return err
			}
		}

		if deviationItem.deviation.IsDownloadable {
			if err := m.downloadDeviationDevAPI(deviationItem, &itemDownloadLog); err != nil {
				return err
			}
		}

		if len(deviationItem.deviation.Videos) > 0 {
			if err := m.downloadVideo(deviationItem); err != nil {
				return err
			}
		}

		if deviationItem.deviation.Flash != nil {
			if err := m.downloadFlash(deviationItem, &itemDownloadLog); err != nil {
				return err
			}
		}

		if err := m.downloadContent(deviationItem, &itemDownloadLog); err != nil {
			return err
		}

		m.DbIO.UpdateTrackedItem(trackedItem, deviationItem.deviation.PublishedTime)
	}

	return nil
}

func (m *deviantArt) downloadFlash(item downloadQueueItemDevAPI, downloadLog *downloadLog) error {
	// download content is always equal to the flash object in the API response
	if downloadLog.download == "" || (downloadLog.download != "" && filepath.Ext(downloadLog.download) != ".swf") {
		return m.daAPI.DownloadFile(
			path.Join(viper.GetString("download.directory"),
				m.Key,
				item.downloadTag,
				fmt.Sprintf(
					"%s_f_%s.swf",
					item.deviation.PublishedTime,
					strings.ReplaceAll(fp.SanitizePath(item.deviation.Title, false), " ", "_"),
				),
			), item.deviation.Flash.Src,
		)
	}

	return nil
}

func (m *deviantArt) downloadVideo(item downloadQueueItemDevAPI) error {
	var (
		biggestFileSize int
		biggestVideo    string
	)

	for _, video := range item.deviation.Videos {
		if video.FileSize > biggestFileSize {
			biggestVideo = video.Src
		}
	}

	return m.daAPI.DownloadFile(
		path.Join(viper.GetString("download.directory"),
			m.Key,
			item.downloadTag,
			fmt.Sprintf(
				"%s_c_%s%s",
				item.deviation.PublishedTime,
				strings.ReplaceAll(fp.SanitizePath(item.deviation.Title, false), " ", "_"),
				fp.GetFileExtension(biggestVideo),
			),
		), biggestVideo,
	)
}

func (m *deviantArt) downloadContent(item downloadQueueItemDevAPI, downloadLog *downloadLog) error {
	if item.deviation.DeviationDownload != nil && item.deviation.Content != nil {
		tmpFile, err := os.CreateTemp("", ".*")
		defer raven.CheckFileRemoval(tmpFile)
		if err != nil {
			return err
		}

		if err = m.daAPI.DownloadFile(tmpFile.Name(), item.deviation.Content.Src); err != nil {
			return err
		}

		sim, err := duplication.CheckForSimilarity(downloadLog.download, tmpFile.Name())
		// if either the file couldn't be converted (probably different file type) or similarity is below 95%
		if err != nil && sim <= 0.95 {
			downloadLog.content, _ = filepath.Abs(path.Join(viper.GetString("download.directory"),
				m.Key,
				item.downloadTag,
				fmt.Sprintf(
					"%s_c_%s%s",
					item.deviation.PublishedTime,
					strings.ReplaceAll(fp.SanitizePath(item.deviation.Title, false), " ", "_"),
					fp.GetFileExtension(item.deviation.Content.Src),
				),
			))
			if err = watcherIO.CopyFile(tmpFile.Name(), downloadLog.content); err != nil {
				return err
			}
		}
	} else if item.deviation.Content != nil {
		downloadLog.content, _ = filepath.Abs(path.Join(viper.GetString("download.directory"),
			m.Key,
			item.downloadTag,
			fmt.Sprintf(
				"%s_c_%s%s",
				item.deviation.PublishedTime,
				strings.ReplaceAll(fp.SanitizePath(item.deviation.Title, false), " ", "_"),
				fp.GetFileExtension(item.deviation.Content.Src),
			),
		))
		if err := m.daAPI.DownloadFile(downloadLog.content, item.deviation.Content.Src); err != nil {
			return err
		}
	}

	return m.downloadThumbs(item, downloadLog)
}

func (m *deviantArt) downloadThumbs(item downloadQueueItemDevAPI, downloadLog *downloadLog) error {
	// compare thumb and download/content
	if len(item.deviation.Thumbs) == 0 {
		return nil
	}

	lastThumb := item.deviation.Thumbs[len(item.deviation.Thumbs)-1]

	// general thumb url across multiple authors for story items which doesn't exist anymore
	if lastThumb.Src == "https://img00.deviantart.net/bb46/a/shared/poetry.jpg" {
		return nil
	}

	tmpFile, err := os.CreateTemp("", ".*")
	defer raven.CheckFileRemoval(tmpFile)
	if err != nil {
		return err
	}

	if err = m.daAPI.DownloadFile(tmpFile.Name(), lastThumb.Src); err != nil {
		return err
	}

	if len(downloadLog.downloadedFiles()) == 0 {
		if err = watcherIO.CopyFile(tmpFile.Name(), path.Join(viper.GetString("download.directory"),
			m.Key,
			item.downloadTag,
			fmt.Sprintf(
				"%s_tmb_%s%s",
				item.deviation.PublishedTime,
				strings.ReplaceAll(fp.SanitizePath(item.deviation.Title, false), " ", "_"),
				fp.GetFileExtension(lastThumb.Src),
			),
		)); err != nil {
			return err
		}

		return nil
	}

	for _, downloadedItem := range downloadLog.downloadedFiles() {
		sim, _ := duplication.CheckForSimilarity(downloadedItem, tmpFile.Name())
		// if either the file couldn't be converted (probably different file type) or similarity is below 95%
		if sim > 0.95 {
			log.WithField("module", m.Key).Debugf(
				"thumbnail matching with %s above threshold %.2f%%", downloadedItem, sim*100,
			)

			return nil
		}
	}

	return watcherIO.CopyFile(tmpFile.Name(), path.Join(viper.GetString("download.directory"),
		m.Key,
		item.downloadTag,
		fmt.Sprintf(
			"%s_tmb_%s%s",
			item.deviation.PublishedTime,
			strings.ReplaceAll(fp.SanitizePath(item.deviation.Title, false), " ", "_"),
			fp.GetFileExtension(lastThumb.Src),
		)),
	)
}

func (m *deviantArt) downloadDeviationDevAPI(item downloadQueueItemDevAPI, downloadLog *downloadLog) error {
	deviationDownload, err := m.daAPI.DeviationDownloadFallback(item.deviation.DeviationURL)
	if err != nil {
		deviationDownload, err = m.daAPI.DeviationDownload(item.deviation.DeviationID)
		if err != nil {
			return err
		}
	}

	item.deviation.DeviationDownload = deviationDownload
	downloadLog.download, _ = filepath.Abs(path.Join(viper.GetString("download.directory"),
		m.Key,
		item.downloadTag,
		fmt.Sprintf(
			"%s_d_%s%s",
			item.deviation.PublishedTime,
			strings.ReplaceAll(fp.SanitizePath(item.deviation.Title, false), " ", "_"),
			fp.GetFileExtension(deviationDownload.Src),
		),
	))

	if err = m.daAPI.DownloadFile(
		downloadLog.download,
		deviationDownload.Src,
	); err != nil {
		return err
	}

	return nil
}

func (m *deviantArt) downloadHTMLContent(item downloadQueueItemDevAPI) error {
	// deviation has text so we retrieve the full content
	deviationContent, err := m.daAPI.DeviationContent(item.deviation.DeviationID)
	if err != nil {
		return err
	}

	text, err := html2text.FromString(deviationContent.HTML)
	if err != nil {
		return err
	}

	filePath := path.Join(viper.GetString("download.directory"),
		m.Key,
		item.downloadTag,
		fmt.Sprintf(
			"%s_t_%s.txt",
			item.deviation.PublishedTime,
			strings.ReplaceAll(fp.SanitizePath(item.deviation.Title, false), " ", "_"),
		),
	)
	if err = os.WriteFile(filePath, []byte(text), os.ModePerm); err != nil {
		return err
	}

	return nil
}
