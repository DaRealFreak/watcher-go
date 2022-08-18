package deviantart

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules/deviantart/napi"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
	"github.com/DaRealFreak/watcher-go/pkg/imaging/duplication"
	"github.com/jaytaylor/html2text"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type downloadQueueItemNAPI struct {
	itemID      string
	deviation   *napi.Deviation
	downloadTag string
}

const downloadQueueItemNAPIDownloadFile = 1
const downloadQueueItemNAPIContentFile = 2

func (i *downloadQueueItemNAPI) GetFileName(fileType int) (filename string) {
	switch fileType {
	case downloadQueueItemNAPIDownloadFile:
		return fmt.Sprintf(
			"%s_%s_d_%s%s",
			i.deviation.GetPublishedTimestamp(),
			i.deviation.DeviationId.String(),
			fp.SanitizePath(i.deviation.GetPrettyName(), false),
			fp.GetFileExtension(i.deviation.Extended.Download.URL),
		)
	case downloadQueueItemNAPIContentFile:
		return fmt.Sprintf(
			"%s_%s_c_%s%s",
			i.deviation.GetPublishedTimestamp(),
			i.deviation.DeviationId.String(),
			fp.SanitizePath(i.deviation.GetPrettyName(), false),
			fp.GetFileExtension(i.deviation.Media.BaseUri),
		)
	}

	return filename
}

func (m *deviantArt) processDownloadQueueNapi(downloadQueue []downloadQueueItemNAPI, trackedItem *models.TrackedItem) error {
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

		if err := m.downloadDeviationNapi(trackedItem, deviationItem); err != nil {
			return err
		}
	}

	return nil
}

func (m *deviantArt) downloadDeviationNapi(trackedItem *models.TrackedItem, deviationItem downloadQueueItemNAPI) error {
	deviationId, _ := strconv.ParseInt(deviationItem.deviation.DeviationId.String(), 10, 64)
	deviationType := napi.DeviationTypeArt
	if deviationItem.deviation.IsJournal {
		deviationType = napi.DeviationTypeJournal
	}

	res, err := m.nAPI.ExtendedDeviation(int(deviationId), deviationItem.deviation.Author.Username, deviationType, false)
	if err != nil {
		return err
	}

	if res.Deviation.PremiumFolderData != nil && !res.Deviation.PremiumFolderData.HasAccess {
		if res.Deviation.PremiumFolderData.Type == napi.PremiumFolderDataWatcherType && m.settings.Download.FollowForContent {
			watchRes, watchErr := m.nAPI.WatchUser(res.Deviation.Author.Username)
			if watchErr != nil {
				return watchErr
			}

			if watchRes.Success {
				log.WithField("module", m.Key).Info(
					fmt.Sprintf("followed user \"%s\" for deviation", res.Deviation.Author.Username),
				)
			} else {
				return fmt.Errorf("unable to follow user \"%s\" for deviation, skipping", res.Deviation.Author.Username)
			}

			if err = m.downloadDeviationNapi(trackedItem, deviationItem); err != nil {
				return err
			}

			if m.settings.Download.UnfollowAfterDownload {
				watchRes, watchErr = m.nAPI.UnwatchUser(res.Deviation.Author.Username)
				if watchErr != nil {
					return watchErr
				}

				if watchRes.Success {
					log.WithField("module", m.Key).Info(
						fmt.Sprintf("unfollowed user \"%s\" after downloading deviation", res.Deviation.Author.Username),
					)
				} else {
					return fmt.Errorf("unable to unfollow user \"%s\" for deviation, skipping", res.Deviation.Author.Username)
				}
			}

			return nil
		}

		log.WithField("module", m.Key).Warnf(
			fmt.Sprintf(
				"no access to deviation \"%s\", deviation is only available to %s, skipping",
				deviationItem.deviation.URL,
				res.Deviation.PremiumFolderData.Type,
			),
		)
		return nil
	}

	if res.Deviation != nil {
		// update the deviation with the extended deviation response if exists as extended
		deviationItem.deviation = res.Deviation
	}

	// ensure download directory, needed for only text artists
	m.nAPI.UserSession.EnsureDownloadDirectory(
		path.Join(
			viper.GetString("download.directory"),
			m.Key,
			deviationItem.downloadTag,
			"tmp.txt",
		),
	)

	if deviationItem.deviation.IsDownloadable && deviationItem.deviation.Extended != nil {
		if err = m.nAPI.UserSession.DownloadFile(
			path.Join(viper.GetString("download.directory"),
				m.Key,
				deviationItem.downloadTag,
				deviationItem.GetFileName(downloadQueueItemNAPIDownloadFile),
			), deviationItem.deviation.Extended.Download.URL); err != nil {
			return err
		}
	}

	// handle token if set
	if deviationItem.deviation.Media.Token != nil && deviationItem.deviation.Media.Token.GetToken() != "" {
		fileUri, _ := url.Parse(deviationItem.deviation.Media.BaseUri)
		fragments := fileUri.Query()
		fragments.Set("token", deviationItem.deviation.Media.Token.GetToken())
		fileUri.RawQuery = fragments.Encode()
		deviationItem.deviation.Media.BaseUri = fileUri.String()
	}

	fullViewType := deviationItem.deviation.Media.GetType(napi.MediaTypeFullView)
	if fullViewType != nil {
		fileUri, _ := url.Parse(deviationItem.deviation.Media.BaseUri)
		fileUri.Path += fullViewType.GetCrop(deviationItem.deviation.Media.PrettyName)
		deviationItem.deviation.Media.BaseUri = fileUri.String()
	}

	// download description if above the min length
	if err = m.downloadDescriptionNapi(deviationItem); err != nil {
		return err
	}

	switch deviationItem.deviation.Type {
	case "journal", "literature":
		if err = m.downloadLiteratureNapi(deviationItem); err != nil {
			return err
		}
		break
	case "image", "pdf", "film", "status":
		if err = m.downloadContentNapi(deviationItem); err != nil {
			return err
		}
		break
	default:
		return fmt.Errorf("unknown deviation type: \"%s\"", deviationItem.deviation.Type)
	}

	m.DbIO.UpdateTrackedItem(trackedItem, deviationItem.itemID)

	return nil
}

func (m *deviantArt) downloadContentNapi(deviationItem downloadQueueItemNAPI) error {
	if highestQualityVideoType := deviationItem.deviation.Media.GetHighestQualityVideoType(); highestQualityVideoType != nil {
		if err := m.nAPI.UserSession.DownloadFile(
			path.Join(viper.GetString("download.directory"),
				m.Key,
				deviationItem.downloadTag,
				fmt.Sprintf(
					"%s_%s_v_%s%s",
					deviationItem.deviation.GetPublishedTimestamp(),
					deviationItem.deviation.DeviationId.String(),
					fp.SanitizePath(deviationItem.deviation.GetPrettyName(), false),
					fp.GetFileExtension(*highestQualityVideoType.URL),
				),
			), *highestQualityVideoType.URL); err != nil {
			return err
		}
	}

	contentFilePath, _ := filepath.Abs(
		path.Join(viper.GetString("download.directory"),
			m.Key,
			deviationItem.downloadTag,
			deviationItem.GetFileName(downloadQueueItemNAPIContentFile),
		),
	)

	fullViewType := deviationItem.deviation.Media.GetType(napi.MediaTypeFullView)
	if fullViewType == nil {
		return nil
	}

	downloadedContentFile := false

	// either the item is not downloadable or it has a different file size to download the full view (or no file size response)
	if !deviationItem.deviation.IsDownloadable ||
		deviationItem.deviation.Extended == nil ||
		(deviationItem.deviation.IsDownloadable &&
			deviationItem.deviation.Extended.Download.FileSize.String() != fullViewType.FileSize.String() &&
			fullViewType.FileSize.String() != "" &&
			fullViewType.FileSize.String() != "0") {
		downloadedContentFile = true
		if err := m.nAPI.UserSession.DownloadFile(contentFilePath, deviationItem.deviation.Media.BaseUri); err != nil {
			return err
		}
	}

	// image comparison if we downloaded the content file and the deviation is downloadable
	if downloadedContentFile &&
		deviationItem.deviation.IsDownloadable &&
		deviationItem.deviation.Extended != nil &&
		fp.GetFileExtension(deviationItem.deviation.Extended.Download.URL) != ".mp4" {
		downloadFilePath, _ := filepath.Abs(
			path.Join(viper.GetString("download.directory"),
				m.Key,
				deviationItem.downloadTag,
				deviationItem.GetFileName(downloadQueueItemNAPIContentFile),
			),
		)

		sim, err := duplication.CheckForSimilarity(downloadFilePath, contentFilePath)
		// if either the file couldn't be converted (probably different file type) or similarity is below 95%
		if err == nil && sim >= 0.95 {
			log.WithField("module", m.Key).Debug(
				fmt.Sprintf(`content has higher match between download and content than configured, removing file %f`, sim),
			)
			return os.Remove(contentFilePath)
		}
	}

	return nil
}

func (m *deviantArt) downloadDescriptionNapi(deviationItem downloadQueueItemNAPI) error {
	// if we couldn't retrieve the extended response we can't access the markup anyway
	if deviationItem.deviation.Extended == nil {
		return nil
	}

	text, err := html2text.FromString(deviationItem.deviation.Extended.DescriptionText.Html.Markup)
	if err != nil {
		return err
	}

	if len(text) > m.settings.Download.DescriptionMinLength {
		filePath := path.Join(viper.GetString("download.directory"),
			m.Key,
			deviationItem.downloadTag,
			fmt.Sprintf(
				"%s_%s_td_%s.txt",
				deviationItem.deviation.GetPublishedTimestamp(),
				deviationItem.deviation.DeviationId.String(),
				fp.SanitizePath(deviationItem.deviation.GetPrettyName(), false),
			),
		)
		log.WithField("module", m.Key).Debugf("downloading description: \"%s\"", filePath)

		if err = ioutil.WriteFile(filePath, []byte(text), os.ModePerm); err != nil {
			return err
		}
	}

	return nil
}

func (m *deviantArt) downloadLiteratureNapi(deviationItem downloadQueueItemNAPI) error {
	text, err := deviationItem.deviation.GetLiteratureContent()
	if err != nil {
		return err
	}

	filePath := path.Join(viper.GetString("download.directory"),
		m.Key,
		deviationItem.downloadTag,
		fmt.Sprintf(
			"%s_%s_t_%s.txt",
			deviationItem.deviation.GetPublishedTimestamp(),
			deviationItem.deviation.DeviationId.String(),
			fp.SanitizePath(deviationItem.deviation.GetPrettyName(), false),
		),
	)
	log.WithField("module", m.Key).Debugf("downloading literature: \"%s\"", filePath)

	if err = ioutil.WriteFile(filePath, []byte(text), os.ModePerm); err != nil {
		return err
	}

	return nil
}
