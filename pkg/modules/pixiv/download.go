package pixiv

import (
	"archive/zip"
	"bytes"
	"fmt"
	"github.com/DaRealFreak/watcher-go/pkg/animation"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"io/ioutil"
	"path"
	"strings"
)

func (m *pixiv) downloadIllustration(downloadQueueItem *downloadQueueItem) (err error) {
	return m.Session.DownloadFile(
		path.Join(viper.GetString("downloadDirectory"), m.Key(), downloadQueueItem.DownloadTag, downloadQueueItem.FileName),
		downloadQueueItem.FileUri,
	)
}

func (m *pixiv) downloadUgoira(downloadQueueItem *downloadQueueItem) (err error) {
	metadata := m.getUgoiraMetaData(downloadQueueItem.ItemId).UgoiraMetadata
	downloadQueueItem.FileName = strings.TrimSuffix(m.GetFileName(metadata.ZipUrls["medium"]), ".zip") + ".webp"
	downloadQueueItem.FileUri = metadata.ZipUrls["medium"]

	resp, err := m.Session.Get(downloadQueueItem.FileUri)
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	zipReader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return err
	}

	animationData := animation.FileData{}
	for _, zipFile := range zipReader.File {
		frame, err := m.getUgoiraFrame(zipFile.Name, metadata)
		if err != nil {
			return err
		}

		unzippedFileBytes, err := m.readZipFile(zipFile)
		if err != nil {
			return err
		}

		delay, err := frame.Delay.Int64()
		if err != nil {
			return err
		}

		animationData.Frames = append(animationData.Frames, unzippedFileBytes)
		animationData.MsDelays = append(animationData.MsDelays, int(delay))
	}

	fileContent, err := m.animationHelper.CreateAnimationWebp(&animationData)
	if err != nil {
		return err
	}

	filepath := path.Join(viper.GetString("downloadDirectory"), m.Key(), downloadQueueItem.DownloadTag, downloadQueueItem.FileName)
	log.Info(fmt.Sprintf("saving converted animation: %s (frames: %d)", filepath, len(animationData.Frames)))
	if _, err := m.pixivSession.WriteToFile(filepath, fileContent); err != nil {
		return err
	}
	return nil
}
