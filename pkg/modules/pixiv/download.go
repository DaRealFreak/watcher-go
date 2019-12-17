package pixiv

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/DaRealFreak/watcher-go/pkg/imaging/animation"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// downloadUgoira handles the download process of ugoira illustration types
func (m *pixiv) downloadUgoira(illustID int) (err error) {
	apiRes, err := m.mobileAPI.GetUgoiraMetadata(illustID)
	if err != nil {
		return err
	}

	fileName := strings.TrimSuffix(m.GetFileName(apiRes.Metadata.ZipURLs.Medium), ".zip") + ".webp"

	resp, err := m.Session.Get(apiRes.Metadata.ZipURLs.Medium)
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
		frame, err := apiRes.GetUgoiraFrame(zipFile.Name)
		if err != nil {
			return err
		}

		unzippedFileBytes, err := m.readZipFile(zipFile)
		if err != nil {
			return err
		}

		delay, _ := frame.Delay.Int64()

		animationData.Frames = append(animationData.Frames, unzippedFileBytes)
		animationData.MsDelays = append(animationData.MsDelays, int(delay))
	}

	// ToDo: fallback to imaging library of golang of error
	fileContent, err := m.animationHelper.CreateAnimationWebp(&animationData)
	if err != nil {
		return err
	}

	// ToDo: replace downloadTag with real download tag
	filepath := path.Join(viper.GetString("download.directory"), m.Key, "downloadTag", fileName)
	log.WithField("module", m.Key).Debug(
		fmt.Sprintf("saving converted animation: %s (frames: %d)", filepath, len(animationData.Frames)),
	)

	if err := ioutil.WriteFile(filepath, fileContent, os.ModePerm); err != nil {
		return err
	}

	return nil
}
