package deviantart

import (
	"errors"
	"fmt"
	"github.com/DaRealFreak/watcher-go/internal/http"
	"github.com/DaRealFreak/watcher-go/internal/http/tls_session"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules/deviantart/napi"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
	"github.com/DaRealFreak/watcher-go/pkg/imaging/duplication"
	log "github.com/sirupsen/logrus"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"time"
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

func (m *deviantArt) processDownloadQueueNapi(downloadQueue []downloadQueueItemNAPI, trackedItem *models.TrackedItem, notifications ...*models.Notification) error {
	if m.settings.MultiProxy {
		// reset usage and errors from previous galleries
		m.resetProxies()
		err := m.processDownloadQueueMultiProxy(downloadQueue, trackedItem)
		if err != nil {
			// Check if it's a session.StatusCode error
			var scErr tls_session.StatusError
			if errors.As(err, &scErr) {
				// 404 and 400 errors are mostly caused by expired CSRF tokens, not exactly sure where to refresh it
				// reading the home page returns 200 and a CSRF token, but it's invalid for the existing queue
				if scErr.StatusCode == 404 || scErr.StatusCode == 400 {
					log.WithField("module", m.Key).Warnf(
						"error occurred downloading item %s (%s) with multi-proxy: %s, re-login",
						trackedItem.URI, downloadQueue[0].itemID, err.Error(),
					)
					os.Exit(-1)
					if successfulLogin := m.Login(m.nAPI.Account); successfulLogin {
						return m.Parse(trackedItem)
					}
				}
			}

			return err
		}
	} else {
		log.WithField("module", m.Key).Info(
			fmt.Sprintf("found %d new items for uri: %s", len(downloadQueue), trackedItem.URI),
		)

		for _, notification := range notifications {
			log.WithField("module", m.Key).Log(
				notification.Level,
				notification.Message,
			)
		}

		for index, deviationItem := range downloadQueue {
			log.WithField("module", m.Key).Info(
				fmt.Sprintf(
					"downloading updates for uri: %s (%0.2f%%)",
					trackedItem.URI,
					float64(index+1)/float64(len(downloadQueue))*100,
				),
			)

			if err := m.downloadDeviationNapi(trackedItem, deviationItem, nil, true); err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *deviantArt) downloadDeviationNapi(
	trackedItem *models.TrackedItem, deviationItem downloadQueueItemNAPI, downloadSession http.TlsClientSessionInterface, update bool,
) error {
	if downloadSession == nil {
		downloadSession = m.nAPI.UserSession
	}

	// record a single “reference” timestamp:
	started := time.Now()

	// collect paths of every file we download / write:
	var downloadedFiles []string

	deviationId, _ := strconv.ParseInt(deviationItem.deviation.DeviationId.String(), 10, 64)
	deviationType := napi.DeviationTypeArt
	if deviationItem.deviation.IsJournal {
		deviationType = napi.DeviationTypeJournal
	}

	res, err := m.nAPI.ExtendedDeviation(int(deviationId), deviationItem.deviation.Author.Username, deviationType, false, downloadSession)
	if err != nil {
		return err
	}

	if res.Deviation.PremiumFolderData != nil && !res.Deviation.PremiumFolderData.HasAccess {
		if res.Deviation.PremiumFolderData.Type == napi.PremiumFolderDataWatcherType && m.settings.Download.FollowForContent {
			watchRes, watchErr := m.nAPI.WatchUser(res.Deviation.Author.Username, downloadSession)
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

			if err = m.downloadDeviationNapi(trackedItem, deviationItem, downloadSession, update); err != nil {
				return err
			}

			if m.settings.Download.UnfollowAfterDownload {
				watchRes, watchErr = m.nAPI.UnwatchUser(res.Deviation.Author.Username, downloadSession)
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
			"no access to deviation \"%s\", deviation is only available to %s, skipping",
			deviationItem.deviation.URL,
			res.Deviation.PremiumFolderData.Type,
		)
		return nil
	}

	if res.Deviation != nil {
		// update the deviation with the extended deviation response if exists as extended
		deviationItem.deviation = res.Deviation
	}

	// ensure download directory, needed for only text artists
	downloadSession.EnsureDownloadDirectory(
		path.Join(
			m.GetDownloadDirectory(),
			m.Key,
			deviationItem.downloadTag,
			"tmp.txt",
		),
	)

	// ──────────────────────────────────────────────────────────────
	// download the “downloadable” version (if it exists)
	// ──────────────────────────────────────────────────────────────
	if deviationItem.deviation.IsDownloadable && deviationItem.deviation.Extended != nil {
		dst := path.Join(
			m.GetDownloadDirectory(),
			m.Key,
			deviationItem.downloadTag,
			deviationItem.GetFileName(downloadQueueItemNAPIDownloadFile),
		)
		if err = downloadSession.DownloadFile(dst, deviationItem.deviation.Extended.Download.URL); err != nil {
			return err
		}

		// record that path
		downloadedFiles = append(downloadedFiles, dst)
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

	// ──────────────────────────────────────────────────────────────
	// download any “AdditionalMedia”, which are mostly slides/galleries
	// ──────────────────────────────────────────────────────────────
	for _, additionalMedia := range deviationItem.deviation.Extended.AdditionalMedia {
		if additionalMedia.Media.BaseUri != "" {
			log.WithField("module", m.Key).Debugf(
				"downloading additional media: %s (%s bytes)",
				additionalMedia.Media.BaseUri,
				additionalMedia.FileSize.String(),
			)

			if additionalMedia.Media.Token != nil && additionalMedia.Media.Token.GetToken() != "" {
				fileUri, _ := url.Parse(additionalMedia.Media.BaseUri)
				fragments := fileUri.Query()
				fragments.Set("token", additionalMedia.Media.Token.GetToken())
				fileUri.RawQuery = fragments.Encode()
				additionalMedia.Media.BaseUri = fileUri.String()
			}

			dst := path.Join(
				m.GetDownloadDirectory(),
				m.Key,
				deviationItem.downloadTag,
				fmt.Sprintf(
					"%s_%s_a_%s_%s%s",
					deviationItem.deviation.GetPublishedTimestamp(),
					deviationItem.deviation.DeviationId.String(),
					additionalMedia.Position.String(),
					fp.SanitizePath(additionalMedia.Media.PrettyName, false),
					fp.GetFileExtension(additionalMedia.Media.BaseUri),
				),
			)
			if err = downloadSession.DownloadFile(dst, additionalMedia.Media.BaseUri); err != nil {
				// fallback to full view if the additional media download failed (got 403 multiple times)
				fullViewType = additionalMedia.Media.GetType(napi.MediaTypeFullView)
				if fullViewType != nil {
					fileUri, _ := url.Parse(additionalMedia.Media.BaseUri)
					fileUri.Path += fullViewType.GetCrop(additionalMedia.Media.PrettyName)
					additionalMedia.Media.BaseUri = fileUri.String()

					if err = downloadSession.DownloadFile(dst, additionalMedia.Media.BaseUri); err != nil {
						return err
					}
				} else {
					return err
				}
			}

			// record that path
			downloadedFiles = append(downloadedFiles, dst)
		}
	}

	// ──────────────────────────────────────────────────────────────
	// download the description (text) if long enough
	// ──────────────────────────────────────────────────────────────
	if err = m.downloadDescriptionNapi(deviationItem, &downloadedFiles); err != nil {
		return err
	}

	// ──────────────────────────────────────────────────────────────
	// download the “literature” (for journal/literature types)
	// ──────────────────────────────────────────────────────────────
	switch deviationItem.deviation.Type {
	case "journal", "literature":
		if err = m.downloadLiteratureNapi(deviationItem, &downloadedFiles); err != nil {
			return err
		}
	case "image", "pdf", "film", "status":
		// ──────────────────────────────────────────────────────────────
		// download the “content” (full-view image/video/etc.)
		// ──────────────────────────────────────────────────────────────
		if err = m.downloadContentNapi(deviationItem, downloadSession, &downloadedFiles); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown deviation type: \"%s\"", deviationItem.deviation.Type)
	}

	if update {
		m.DbIO.UpdateTrackedItem(trackedItem, deviationItem.itemID)
	}

	// ──────────────────────────────────────────────────────────────
	// now that all files are downloaded, reset their timestamps
	// ──────────────────────────────────────────────────────────────
	for idx, f := range downloadedFiles {
		// Compute a slightly bumped‐up timestamp for each file to still enable sorting by timestamp
		// (f.e. if we downloaded 10 files, we want them to be able to sort them by timestamp)
		// the smallest unit we can use surprisingly differs on the file system:
		// - ext4: 1 nanosecond
		// - APFS: 1 nanosecond
		// - NTFS: 100 nanoseconds
		// base + idx * 1 millisecond
		t := started.Add(time.Duration(idx) * time.Millisecond)

		if info, infoErr := os.Stat(f); infoErr == nil && !info.IsDir() {
			if err = os.Chtimes(f, t, t); err != nil {
				log.WithField("module", m.Key).Warnf(
					"failed to reset timestamp for %s: %v", f, err,
				)
			}
		}
	}

	return nil
}

func (m *deviantArt) downloadDescriptionNapi(deviationItem downloadQueueItemNAPI, downloadedFiles *[]string) error {
	// if we couldn't retrieve the extended response, we can't access the markup anyway
	if deviationItem.deviation.Extended == nil {
		return nil
	}

	text, err := deviationItem.deviation.Extended.DescriptionText.GetTextContent()
	if err != nil {
		return err
	}

	if len(text) > m.settings.Download.DescriptionMinLength {
		filePath := path.Join(
			m.GetDownloadDirectory(),
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

		if err = os.WriteFile(filePath, []byte(text), os.ModePerm); err != nil {
			return err
		}

		// record that path
		*downloadedFiles = append(*downloadedFiles, filePath)
	}

	return nil
}

func (m *deviantArt) downloadLiteratureNapi(deviationItem downloadQueueItemNAPI, downloadedFiles *[]string) error {
	text, err := deviationItem.deviation.TextContent.GetTextContent()
	if err != nil {
		return err
	}

	filePath := path.Join(
		m.GetDownloadDirectory(),
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

	if err = os.WriteFile(filePath, []byte(text), os.ModePerm); err != nil {
		return err
	}

	// record that path
	*downloadedFiles = append(*downloadedFiles, filePath)
	return nil
}

func (m *deviantArt) downloadContentNapi(
	deviationItem downloadQueueItemNAPI, downloadSession http.TlsClientSessionInterface, downloadedFiles *[]string,
) error {
	if downloadSession == nil {
		downloadSession = m.nAPI.UserSession
	}

	// 1) possibly download a video
	if highestQualityVideoType := deviationItem.deviation.Media.GetHighestQualityVideoType(); highestQualityVideoType != nil {
		dst := path.Join(
			m.GetDownloadDirectory(),
			m.Key,
			deviationItem.downloadTag,
			fmt.Sprintf(
				"%s_%s_v_%s%s",
				deviationItem.deviation.GetPublishedTimestamp(),
				deviationItem.deviation.DeviationId.String(),
				fp.SanitizePath(deviationItem.deviation.GetPrettyName(), false),
				fp.GetFileExtension(*highestQualityVideoType.URL),
			),
		)
		if err := downloadSession.DownloadFile(dst, *highestQualityVideoType.URL); err != nil {
			return err
		}

		// record that path
		*downloadedFiles = append(*downloadedFiles, dst)
	}

	// 2) set up contentFilePath
	contentFilePath, _ := filepath.Abs(
		path.Join(
			m.GetDownloadDirectory(),
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

	// 3) either the item is not downloadable, or sizes differ
	if !deviationItem.deviation.IsDownloadable ||
		deviationItem.deviation.Extended == nil ||
		(deviationItem.deviation.IsDownloadable &&
			deviationItem.deviation.Extended.Download.FileSize.String() != fullViewType.FileSize.String() &&
			fullViewType.FileSize.String() != "" &&
			fullViewType.FileSize.String() != "0") {
		downloadedContentFile = true
		if err := downloadSession.DownloadFile(contentFilePath, deviationItem.deviation.Media.BaseUri); err != nil {
			return err
		}
		// record that path
		*downloadedFiles = append(*downloadedFiles, contentFilePath)
	}

	// 4) check if we have a PDF media type
	// if the deviation is downloadable, it's always the same PDF file, so skip on IsDownloadable
	if !deviationItem.deviation.IsDownloadable {
		if pdfMedia := deviationItem.deviation.Media.GetPdfMedia(); pdfMedia != nil && pdfMedia.Source != nil {
			dst := path.Join(
				m.GetDownloadDirectory(),
				m.Key,
				deviationItem.downloadTag,
				fmt.Sprintf(
					"%s_%s_p_%s.pdf",
					deviationItem.deviation.GetPublishedTimestamp(),
					deviationItem.deviation.DeviationId.String(),
					fp.SanitizePath(deviationItem.deviation.GetPrettyName(), false),
				),
			)
			if err := downloadSession.DownloadFile(dst, *pdfMedia.Source); err != nil {
				return err
			}
			// record that path
			*downloadedFiles = append(*downloadedFiles, dst)
		}
	}

	// 5) If we downloaded “contentFilePath” and it's an image that needs similarity checking:
	if downloadedContentFile &&
		deviationItem.deviation.IsDownloadable &&
		deviationItem.deviation.Extended != nil &&
		fp.GetFileExtension(deviationItem.deviation.Extended.Download.URL) != ".mp4" &&
		fp.GetFileExtension(deviationItem.deviation.Extended.Download.URL) != ".zip" &&
		fp.GetFileExtension(deviationItem.deviation.Extended.Download.URL) != ".pdf" {

		downloadFilePath, _ := filepath.Abs(
			path.Join(
				m.GetDownloadDirectory(),
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
