package pixiv

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io/ioutil"
	"path"
	"strings"

	"github.com/DaRealFreak/watcher-go/pkg/animation"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// downloadIllustration handles the download process of Illustration and Manga illustration types
func (m *pixiv) downloadIllustration(downloadQueueItem *downloadQueueItem) (err error) {
	for i := len(downloadQueueItem.Illustration.MetaPages) - 1; i >= 0; i-- {
		image := downloadQueueItem.Illustration.MetaPages[i]
		fileName := m.GetFileName(image["image_urls"]["original"])
		fileURI := image["image_urls"]["original"]
		if err := m.Session.DownloadFile(
			path.Join(viper.GetString("downloadDirectory"), m.Key(), downloadQueueItem.DownloadTag, fileName),
			fileURI,
		); err != nil {
			// if download was not successful return the occurred error here
			return err
		}
	}

	if len(downloadQueueItem.Illustration.MetaSinglePage) > 0 {
		fileName := m.GetFileName(downloadQueueItem.Illustration.MetaSinglePage["original_image_url"])
		fileURI := downloadQueueItem.Illustration.MetaSinglePage["original_image_url"]
		return m.Session.DownloadFile(
			path.Join(viper.GetString("downloadDirectory"), m.Key(), downloadQueueItem.DownloadTag, fileName),
			fileURI,
		)
	}
	return nil
}

// downloadUgoira handles the download process of ugoira illustration types
func (m *pixiv) downloadUgoira(downloadQueueItem *downloadQueueItem) (err error) {
	metadata := m.getUgoiraMetaData(downloadQueueItem.ItemID).UgoiraMetadata
	fileName := strings.TrimSuffix(m.GetFileName(metadata.ZipUrls["medium"]), ".zip") + ".webp"
	fileURI := metadata.ZipUrls["medium"]

	resp, err := m.Session.Get(fileURI)
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

	// ToDo: fallback to imaging library of golang of error
	fileContent, err := m.animationHelper.CreateAnimationWebp(&animationData)
	if err != nil {
		return err
	}

	filepath := path.Join(viper.GetString("downloadDirectory"), m.Key(), downloadQueueItem.DownloadTag, fileName)
	log.Info(fmt.Sprintf("saving converted animation: %s (frames: %d)", filepath, len(animationData.Frames)))
	if _, err := m.pixivSession.WriteToFile(filepath, fileContent); err != nil {
		return err
	}
	return nil
}
