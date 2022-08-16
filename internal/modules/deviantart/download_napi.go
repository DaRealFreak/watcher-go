package deviantart

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/jaytaylor/html2text"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules/deviantart/napi"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type downloadQueueItemNAPI struct {
	itemID      string
	deviation   *napi.Deviation
	downloadTag string
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

		deviationId, _ := strconv.ParseInt(deviationItem.deviation.DeviationId.String(), 10, 64)
		deviationType := napi.DeviationTypeArt
		if deviationItem.deviation.IsJournal {
			deviationType = napi.DeviationTypeJournal
		}

		res, err := m.nAPI.ExtendedDeviation(int(deviationId), deviationItem.deviation.Author.Username, deviationType, false)
		if err != nil {
			return err
		}

		// update the deviation with the extended deviation response
		deviationItem.deviation = res.Deviation

		// ensure download directory, needed for only text artists
		m.nAPI.UserSession.EnsureDownloadDirectory(
			path.Join(
				viper.GetString("download.directory"),
				m.Key,
				deviationItem.downloadTag,
				"tmp.txt",
			),
		)

		if deviationItem.deviation.IsDownloadable {
			if err = m.nAPI.UserSession.DownloadFile(
				path.Join(viper.GetString("download.directory"),
					m.Key,
					deviationItem.downloadTag,
					fmt.Sprintf(
						"%s_%s_d_%s%s",
						deviationItem.itemID,
						deviationItem.deviation.DeviationId.String(),
						m.SanitizePath(deviationItem.deviation.Media.PrettyName, false),
						m.GetFileExtension(deviationItem.deviation.Extended.Download.URL),
					),
				), deviationItem.deviation.Extended.Download.URL); err != nil {
				return err
			}
		}

		// download description if above the min length
		if err = m.downloadDescriptionNapi(deviationItem); err != nil {
			return err
		}

		switch deviationItem.deviation.Type {
		case "image":
		case "pdf":
			// https://www.deviantart.com/awesomex18/art/The-Infinite-Generosity-of-Goddess-Yoko-614172678
		case "film":
			// https://www.deviantart.com/ourcouncil/art/Softball-Squashing-2-Public-Preview-918834222
			if err = m.downloadContentNapi(deviationItem); err != nil {
				return err
			}
			break
		case "literature":
			if err = m.downloadLiteratureNapi(deviationItem); err != nil {
				return err
			}
			break
		default:
			println("unknown")
		}

		m.DbIO.UpdateTrackedItem(trackedItem, deviationItem.itemID)
	}

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
					deviationItem.itemID,
					deviationItem.deviation.DeviationId.String(),
					m.SanitizePath(deviationItem.deviation.Media.PrettyName, false),
					m.GetFileExtension(*highestQualityVideoType.URL),
				),
			), *highestQualityVideoType.URL); err != nil {
			return err
		}
	}

	if err := m.nAPI.UserSession.DownloadFile(
		path.Join(viper.GetString("download.directory"),
			m.Key,
			deviationItem.downloadTag,
			fmt.Sprintf(
				"%s_%s_c_%s%s",
				deviationItem.itemID,
				deviationItem.deviation.DeviationId.String(),
				m.SanitizePath(deviationItem.deviation.Media.PrettyName, false),
				m.GetFileExtension(deviationItem.deviation.Media.BaseUri),
			),
		), deviationItem.deviation.Media.BaseUri); err != nil {
		return err
	}

	return nil
}

func (m *deviantArt) downloadDescriptionNapi(deviationItem downloadQueueItemNAPI) error {
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
				deviationItem.itemID,
				deviationItem.deviation.DeviationId.String(),
				strings.ReplaceAll(m.SanitizePath(deviationItem.deviation.Title, false), " ", "_"),
			),
		)
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
			deviationItem.itemID,
			deviationItem.deviation.DeviationId.String(),
			strings.ReplaceAll(m.SanitizePath(deviationItem.deviation.Title, false), " ", "_"),
		),
	)
	if err = ioutil.WriteFile(filePath, []byte(text), os.ModePerm); err != nil {
		return err
	}

	return nil
}
