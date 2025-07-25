package pixiv

import (
	"archive/zip"
	"bytes"
	"fmt"
	"github.com/DaRealFreak/watcher-go/pkg/linkfinder"
	"io"
	"net/url"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/http/tls_session"
	"github.com/DaRealFreak/watcher-go/internal/models"
	fanboxapi "github.com/DaRealFreak/watcher-go/internal/modules/pixiv/fanbox_api"
	mobileapi "github.com/DaRealFreak/watcher-go/internal/modules/pixiv/mobile_api"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
	"github.com/DaRealFreak/watcher-go/pkg/imaging/animation"
	log "github.com/sirupsen/logrus"
)

type downloadQueueItem struct {
	ItemID       int
	DownloadTag  string
	DownloadItem interface{}
}

func (m *pixiv) processDownloadQueue(downloadQueue []*downloadQueueItem, trackedItem *models.TrackedItem, notifications ...*models.Notification) error {
	log.WithField("module", m.Key).Info(
		fmt.Sprintf("found %d new items for uri: \"%s\"", len(downloadQueue), trackedItem.URI),
	)

	for _, notification := range notifications {
		log.WithField("module", m.Key).Log(
			notification.Level,
			notification.Message,
		)
	}

	for index, data := range downloadQueue {
		log.WithField("module", m.Key).Info(
			fmt.Sprintf(
				"downloading updates for uri: \"%s\" (%0.2f%%)",
				trackedItem.URI,
				float64(index+1)/float64(len(downloadQueue))*100,
			),
		)

		switch item := data.DownloadItem.(type) {
		case mobileapi.Illustration:
			var err error

			switch item.Type {
			case Ugoira:
				err = m.downloadUgoira(data, item.ID)
			default:
				err = m.downloadIllustration(data, item)
			}

			if m.settings.ExternalUrls.PrintExternalItems {
				checkedIllustration := item
				// I could not find a pattern when the caption is actually empty
				// or just removed in the API Search/UserIllustration response, so we check the actual details
				if item.Caption == "" {
					detail, detailErr := m.mobileAPI.GetIllustDetail(item.ID)
					if detailErr != nil {
						log.WithField("module", m.Key).Errorf(
							"failed to get illustration details for ID %d: %v",
							item.ID,
							detailErr,
						)
					}
					checkedIllustration = detail.Illustration
				}
				links := linkfinder.GetLinks(checkedIllustration.Caption)
				for _, link := range links {
					log.WithField("module", m.Key).Infof(
						"found external URL: \"%s\" in post \"https://www.pixiv.net/en/artworks/%d\"",
						link,
						checkedIllustration.ID,
					)
				}
			}

			if err != nil {
				if e, ok := err.(tls_session.StatusError); ok && e.StatusCode == 404 {
					log.WithField("module", m.ModuleKey()).Warnf(
						"received 404 status code for URI \"https://www.pixiv.net/en/artworks/%d\", "+
							"content got most likely deleted, skipping",
						data.ItemID,
					)
				} else {
					return err
				}
			}

			m.DbIO.UpdateTrackedItem(trackedItem, strconv.Itoa(data.ItemID))
		case fanboxapi.FanboxPost:
			if err := m.downloadFanboxPost(data, item); err != nil {
				return err
			}

			m.DbIO.UpdateTrackedItem(trackedItem, strconv.Itoa(data.ItemID))
		}

		m.DbIO.UpdateTrackedItem(trackedItem, strconv.Itoa(data.ItemID))
	}

	return nil
}

// downloadFanboxPost downloads a pixiv fanbox post after retrieving the post info from the Fanbox API
// nolint: funlen
func (m *pixiv) downloadFanboxPost(data *downloadQueueItem, post fanboxapi.FanboxPost) error {
	postID, err := post.ID.Int64()
	if err != nil {
		return err
	}

	postInfo, err := m.fanboxAPI.GetPostInfo(int(postID))
	if err != nil {
		return err
	}

	pattern := regexp.MustCompile(`(?m)https?://[-a-zA-Z0-9@:%._+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_+.~#?&/=]*)`)
	for _, comment := range postInfo.CommentsFromAuthor() {
		urlMatches := pattern.FindAllStringSubmatch(comment, -1)
		if len(urlMatches) > 0 {
			for _, match := range urlMatches {
				q, _ := url.Parse(match[0])
				log.WithField("module", m.Key).Warnf("found URL from author in comments: %s (%d)", q.String(), postID)
			}
		}
	}

	urlMatches := pattern.FindAllStringSubmatch(postInfo.Body.PostBody.Text, -1)
	if len(urlMatches) > 0 {
		for _, match := range urlMatches {
			q, _ := url.Parse(match[0])
			log.WithField("module", m.Key).Warnf("found URL from author in text body: %s (%d)", q.String(), postID)
		}
	}

	if postInfo.Body.ImageForShare != "" {
		fileName := fmt.Sprintf("0_%s", fp.GetFileName(postInfo.Body.ImageForShare))

		if err = m.fanboxAPI.DownloadFile(
			path.Join(
				m.GetDownloadDirectory(), m.Key, data.DownloadTag, postInfo.Body.ID.String(), fileName,
			),
			postInfo.Body.ImageForShare,
		); err != nil {
			return err
		}
	}

	for i, image := range postInfo.Body.PostBody.Images {
		fileName := fmt.Sprintf("%d_%s", i+1, fp.GetFileName(image.OriginalURL))

		if err = m.fanboxAPI.DownloadFile(
			path.Join(
				m.GetDownloadDirectory(), m.Key, data.DownloadTag, postInfo.Body.ID.String(), fileName,
			),
			image.OriginalURL,
		); err != nil {
			// if download was not successful return the occurred error here
			return err
		}
	}

	for i, file := range postInfo.Body.PostBody.Files {
		fileName := fmt.Sprintf("%d_%s.%s", i+1, file.Name, file.Extension)

		if err = m.fanboxAPI.DownloadFile(
			path.Join(
				m.GetDownloadDirectory(), m.Key, data.DownloadTag, postInfo.Body.ID.String(), fileName,
			),
			file.URL,
		); err != nil {
			// if download was not successful return the occurred error here
			return err
		}
	}

	var i = 0
	for _, file := range postInfo.Body.PostBody.FileMap {
		i += 1
		fileName := fmt.Sprintf("%d_%s.%s", i, file.Name, file.Extension)

		if err = m.fanboxAPI.DownloadFile(
			path.Join(
				m.GetDownloadDirectory(), m.Key, data.DownloadTag, postInfo.Body.ID.String(), fileName,
			),
			file.URL,
		); err != nil {
			// if download was not successful return the occurred error here
			return err
		}
	}

	for i, file := range postInfo.ImagesFromBlocks() {
		fileName := fmt.Sprintf("%d_%s", i+1, fp.GetFileName(file))

		if err = m.fanboxAPI.DownloadFile(
			path.Join(
				m.GetDownloadDirectory(), m.Key, data.DownloadTag, postInfo.Body.ID.String(), fileName,
			),
			file,
		); err != nil {
			// if download was not successful return the occurred error here
			return err
		}
	}

	return nil
}

func (m *pixiv) downloadIllustration(data *downloadQueueItem, illust mobileapi.Illustration) error {
	for _, metaPage := range illust.MetaPages {
		fileName := fp.GetFileName(metaPage.ImageURLs.Original)

		if err := m.mobileAPI.DownloadFile(
			path.Join(m.GetDownloadDirectory(), m.Key, data.DownloadTag, fileName),
			metaPage.ImageURLs.Original,
		); err != nil {
			// if download was not successful return the occurred error here
			return err
		}
	}

	if illust.MetaSinglePage.OriginalImageURL != nil {
		fileName := fp.GetFileName(*illust.MetaSinglePage.OriginalImageURL)

		return m.mobileAPI.DownloadFile(
			path.Join(m.GetDownloadDirectory(), m.Key, data.DownloadTag, fileName),
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
		strings.TrimSuffix(fp.GetFileName(apiRes.Metadata.ZipURLs.Medium), ".zip"),
		m.settings.Animation.Format,
	)

	resp, err := m.mobileAPI.Get(apiRes.Metadata.ZipURLs.Medium)
	if err != nil {
		return err
	}

	body, err := io.ReadAll(resp.Body)
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
	case animation.FileFormatWebm:
		fileContent, err = m.animationHelper.CreateAnimationWebM(&animationData)
	case animation.FileFormatWebp:
		fileContent, err = m.animationHelper.CreateAnimationWebp(&animationData)
	case animation.FileFormatGif:
		fileContent, err = m.animationHelper.CreateAnimationGif(&animationData)
	default:
		fileContent, err = m.animationHelper.CreateAnimationWebM(&animationData)
	}

	if err != nil && m.settings.Animation.LowQualityGifFallback {
		fileContent, err = m.animationHelper.CreateAnimationGifGo(&animationData)
	}

	if err != nil {
		return err
	}

	filepath := path.Join(m.GetDownloadDirectory(), m.Key, data.DownloadTag, fileName)
	log.WithField("module", m.Key).Debug(
		fmt.Sprintf("saving converted animation: %s (frames: %d)", filepath, len(animationData.Frames)),
	)

	m.mobileAPI.Session.EnsureDownloadDirectory(filepath)

	if err = os.WriteFile(filepath, fileContent, os.ModePerm); err != nil {
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
