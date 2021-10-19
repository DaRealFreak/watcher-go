package pixiv

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/models"
	ajaxapi "github.com/DaRealFreak/watcher-go/internal/modules/pixiv/ajax_api"
	mobileapi "github.com/DaRealFreak/watcher-go/internal/modules/pixiv/mobile_api"
	publicapi "github.com/DaRealFreak/watcher-go/internal/modules/pixiv/public_api"
	"github.com/DaRealFreak/watcher-go/pkg/imaging/animation"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type downloadQueueItem struct {
	ItemID       int
	DownloadTag  string
	DownloadItem interface{}
}

func (m *pixiv) processDownloadQueue(downloadQueue []*downloadQueueItem, trackedItem *models.TrackedItem) error {
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

		switch item := data.DownloadItem.(type) {
		case publicapi.PublicIllustration:
			if err := m.downloadPublicIllustration(data, item); err != nil {
				return err
			}

			m.DbIO.UpdateTrackedItem(trackedItem, strconv.Itoa(data.ItemID))
		case mobileapi.Illustration:
			var err error

			switch item.Type {
			case Ugoira:
				err = m.downloadUgoira(data, item.ID)
			default:
				err = m.downloadIllustration(data, item)
			}

			if err != nil {
				return err
			}

			m.DbIO.UpdateTrackedItem(trackedItem, strconv.Itoa(data.ItemID))
		case ajaxapi.FanboxPost:
			if err := m.downloadFanboxPost(data, item); err != nil {
				return err
			}

			m.DbIO.UpdateTrackedItem(trackedItem, strconv.Itoa(data.ItemID))
		}

		m.DbIO.UpdateTrackedItem(trackedItem, strconv.Itoa(data.ItemID))
	}

	return nil
}

// downloadFanboxPost downloads a pixiv fanbox post after retrieving the post info from the AJAX API
// nolint: funlen
func (m *pixiv) downloadFanboxPost(data *downloadQueueItem, post ajaxapi.FanboxPost) error {
	postID, err := post.ID.Int64()
	if err != nil {
		return err
	}

	postInfo, err := m.ajaxAPI.GetPostInfo(int(postID))
	if err != nil {
		return err
	}

	if postInfo.Body.ImageForShare != "" {
		fileName := fmt.Sprintf("0_%s", m.GetFileName(postInfo.Body.ImageForShare))

		if err := m.ajaxAPI.Session.DownloadFile(
			path.Join(
				viper.GetString("download.directory"), m.Key, data.DownloadTag, postInfo.Body.ID.String(), fileName,
			),
			postInfo.Body.ImageForShare,
		); err != nil {
			return err
		}
	}

	for i, image := range postInfo.Body.PostBody.Images {
		fileName := fmt.Sprintf("%d_%s", i+1, m.GetFileName(image.OriginalURL))

		if err := m.ajaxAPI.Session.DownloadFile(
			path.Join(
				viper.GetString("download.directory"), m.Key, data.DownloadTag, postInfo.Body.ID.String(), fileName,
			),
			image.OriginalURL,
		); err != nil {
			// if download was not successful return the occurred error here
			return err
		}
	}

	for i, file := range postInfo.Body.PostBody.Files {
		fileName := fmt.Sprintf("%d_%s.%s", i+1, file.Name, file.Extension)

		if err := m.ajaxAPI.Session.DownloadFile(
			path.Join(
				viper.GetString("download.directory"), m.Key, data.DownloadTag, postInfo.Body.ID.String(), fileName,
			),
			file.URL,
		); err != nil {
			// if download was not successful return the occurred error here
			return err
		}
	}

	for i, file := range postInfo.ImagesFromBlocks() {
		fileName := fmt.Sprintf("%d_%s", i+1, m.GetFileName(file))

		if err := m.ajaxAPI.Session.DownloadFile(
			path.Join(
				viper.GetString("download.directory"), m.Key, data.DownloadTag, postInfo.Body.ID.String(), fileName,
			),
			file,
		); err != nil {
			// if download was not successful return the occurred error here
			return err
		}
	}

	return nil
}

func (m *pixiv) downloadPublicIllustration(data *downloadQueueItem, illust publicapi.PublicIllustration) error {
	switch {
	case illust.PageCount > 1:
		illustration, err := m.mobileAPI.GetIllustDetail(illust.ID)
		if err != nil {
			return err
		}

		return m.downloadIllustration(data, illustration.Illustration)
	case illust.Type == Ugoira:
		return m.downloadUgoira(data, illust.ID)
	default:
		if err := m.mobileAPI.Session.DownloadFile(
			path.Join(
				viper.GetString("download.directory"),
				m.Key,
				data.DownloadTag,
				m.GetFileName(illust.ImageURLs.Large),
			),
			illust.ImageURLs.Large,
		); err != nil {
			return err
		}
	}

	return nil
}

func (m *pixiv) downloadIllustration(data *downloadQueueItem, illust mobileapi.Illustration) error {
	for _, metaPage := range illust.MetaPages {
		fileName := m.GetFileName(metaPage.ImageURLs.Original)

		if err := m.mobileAPI.Session.DownloadFile(
			path.Join(viper.GetString("download.directory"), m.Key, data.DownloadTag, fileName),
			metaPage.ImageURLs.Original,
		); err != nil {
			// if download was not successful return the occurred error here
			return err
		}
	}

	if illust.MetaSinglePage.OriginalImageURL != nil {
		fileName := m.GetFileName(*illust.MetaSinglePage.OriginalImageURL)

		return m.mobileAPI.Session.DownloadFile(
			path.Join(viper.GetString("download.directory"), m.Key, data.DownloadTag, fileName),
			*illust.MetaSinglePage.OriginalImageURL,
		)
	}

	return nil
}

// downloadUgoira handles the download process of ugoira illustration types
func (m *pixiv) downloadUgoira(data *downloadQueueItem, illustID int) (err error) {
	apiRes, err := m.mobileAPI.GetUgoiraMetadata(illustID)
	if err != nil {
		return err
	}

	fileName := fmt.Sprintf(
		"%s%s",
		strings.TrimSuffix(m.GetFileName(apiRes.Metadata.ZipURLs.Medium), ".zip"),
		m.settings.Animation.Format,
	)

	resp, err := m.mobileAPI.Session.Get(apiRes.Metadata.ZipURLs.Medium)
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

	animationData, err := m.getAnimationData(zipReader, apiRes)
	if err != nil {
		return err
	}

	var fileContent []byte

	switch m.settings.Animation.Format {
	case animation.FileFormatWebp:
		fileContent, err = m.animationHelper.CreateAnimationWebp(&animationData)
	case animation.FileFormatGif:
		fileContent, err = m.animationHelper.CreateAnimationGif(&animationData)
	default:
		fileContent, err = m.animationHelper.CreateAnimationWebp(&animationData)
	}

	if err != nil && m.settings.Animation.LowQualityGifFallback {
		fileContent, err = m.animationHelper.CreateAnimationGifGo(&animationData)
	}

	if err != nil {
		return err
	}

	filepath := path.Join(viper.GetString("download.directory"), m.Key, data.DownloadTag, fileName)
	log.WithField("module", m.Key).Debug(
		fmt.Sprintf("saving converted animation: %s (frames: %d)", filepath, len(animationData.Frames)),
	)

	m.mobileAPI.Session.EnsureDownloadDirectory(filepath)

	if err := ioutil.WriteFile(filepath, fileContent, os.ModePerm); err != nil {
		return err
	}

	return nil
}

func (m *pixiv) getAnimationData(zipReader *zip.Reader, apiRes *mobileapi.UgoiraMetadata) (animation.FileData, error) {
	animationData := animation.FileData{}

	for _, zipFile := range zipReader.File {
		frame, err := apiRes.GetUgoiraFrame(zipFile.Name)
		if err != nil {
			return animation.FileData{}, err
		}

		unzippedFileBytes, err := m.readZipFile(zipFile)
		if err != nil {
			return animation.FileData{}, err
		}

		delay, _ := frame.Delay.Int64()

		animationData.Frames = append(animationData.Frames, unzippedFileBytes)
		animationData.MsDelays = append(animationData.MsDelays, int(delay))
	}

	return animationData, nil
}
