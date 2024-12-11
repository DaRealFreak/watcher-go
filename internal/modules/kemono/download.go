package kemono

import (
	"fmt"
	"github.com/DaRealFreak/watcher-go/internal/modules/kemono/api"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/http/session"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
	"github.com/PuerkitoBio/goquery"
	log "github.com/sirupsen/logrus"
)

func (m *kemono) processDownloadQueue(item *models.TrackedItem, downloadQueue []api.Result, notifications ...*models.Notification) error {
	log.WithField("module", m.Key).Info(
		fmt.Sprintf("found %d new items for uri: \"%s\"", len(downloadQueue), item.URI),
	)

	if notifications != nil {
		for _, notification := range notifications {
			log.WithField("module", m.Key).Log(
				notification.Level,
				notification.Message,
			)
		}
	}

	for index, data := range downloadQueue {
		log.WithField("module", m.Key).Info(
			fmt.Sprintf(
				"downloading updates for uri: \"%s\" (%0.2f%%)",
				item.URI,
				float64(index+1)/float64(len(downloadQueue))*100,
			),
		)

		if err := m.downloadPost(item, data); err != nil {
			return err
		}

		m.DbIO.UpdateTrackedItem(item, data.ID)
	}

	return nil
}

func (m *kemono) downloadPost(item *models.TrackedItem, data api.Result) error {
	webUrl := fmt.Sprintf("%s/%s/user/%s/post/%s", m.baseUrl.String(), data.Service, data.User, data.ID)
	post, err := m.api.GetPostDetails(data.Service, data.User, data.ID)
	if err != nil {
		return fmt.Errorf("failed to fetch post details: %w", err)
	}

	postComments, commentErr := m.api.GetPostComments(data.Service, data.User, data.ID)
	if commentErr != nil {
		return fmt.Errorf("failed to fetch post comments: %w", commentErr)
	}

	for index, downloadItem := range m.getDownloadLinks(post) {
		parsedLink, parsedErr := url.Parse(downloadItem.FileURI)
		if parsedErr != nil {
			return parsedErr
		}

		fileName := fp.SanitizePath(fp.GetFileName(downloadItem.FileURI), false)
		if parsedLink.Query().Get("f") != "" {
			fileName = fp.SanitizePath(parsedLink.Query().Get("f"), false)
		}

		// ignore mega folder icons
		if strings.Contains(fileName, "mega.nz_rich-folder") {
			continue
		}

		postFolderPath := fp.SanitizePath(data.ID, false)
		sanitizedPostTitle := fp.SanitizePath(data.Title, false)
		if strings.TrimSpace(sanitizedPostTitle) != "" {
			postFolderPath += " - " + sanitizedPostTitle
		}

		if err = m.Session.DownloadFile(
			path.Join(
				m.GetDownloadDirectory(),
				m.Key,
				fp.TruncateMaxLength(m.getSubFolder(item)),
				fp.TruncateMaxLength(postFolderPath),
				fp.TruncateMaxLength(strings.TrimSpace(fmt.Sprintf("%s_%d_%s", data.ID, index+1, fileName))),
			),
			downloadItem.FileURI,
		); err != nil {
			if e, ok := err.(session.StatusError); ok && e.StatusCode == 404 && downloadItem.FallbackFileURI != "" {
				log.WithField("module", m.Key).Warnf(
					"received 404 status code error \"%s\", trying thumbnail fallback url \"%s\"",
					err.Error(),
					downloadItem.FallbackFileURI,
				)

				if err = m.Session.DownloadFile(
					path.Join(
						m.GetDownloadDirectory(),
						m.Key,
						fp.TruncateMaxLength(m.getSubFolder(item)),
						fp.TruncateMaxLength(fp.SanitizePath(data.ID, false)),
						fp.TruncateMaxLength(strings.TrimSpace(fmt.Sprintf("%s_%d_fallback_%s", data.ID, index+1, fileName))),
					),
					downloadItem.FallbackFileURI,
				); err != nil {
					return err
				}
			}

			return err
		}
	}

	factory := modules.GetModuleFactory()
	for _, externalURL := range m.getExternalLinks(post.Post.Content, postComments) {
		if m.settings.ExternalURLs.PrintExternalItems {
			log.WithField("module", m.Key).Infof(
				"found external URL: \"%s\" in post \"%s\"",
				externalURL,
				webUrl,
			)
		}

		if m.settings.ExternalURLs.DownloadExternalItems {
			if factory.CanParse(externalURL) {
				module := modules.GetModuleFactory().GetModuleFromURI(externalURL)
				if err = module.Load(); err != nil {
					return err
				}
				newItem := m.DbIO.GetFirstOrCreateTrackedItem(externalURL, "", module)
				// don't delete previously already added items
				deleteAfter := newItem.CurrentItem == ""
				if m.Cfg.Run.ForceNew && newItem.CurrentItem != "" {
					log.WithField("module", m.Key).Info(
						fmt.Sprintf("resetting progress for item %s (current id: %s)", newItem.URI, newItem.CurrentItem),
					)
					newItem.CurrentItem = ""
					m.DbIO.ChangeTrackedItemCompleteStatus(newItem, false)
					m.DbIO.UpdateTrackedItem(newItem, "")
				}

				if err = module.Parse(newItem); err != nil {
					log.WithField("module", m.Key).Warnf(
						"unable to parse external URL \"%s\" found in post \"%s\" with error \"%s\", skipping",
						newItem.URI,
						webUrl,
						err.Error(),
					)
					if !m.settings.ExternalURLs.SkipErrorsForExternalURLs {
						if deleteAfter {
							m.DbIO.DeleteTrackedItem(newItem)
						}
						return err
					}
				}

				// delete newly created item after we parsed it
				if deleteAfter {
					m.DbIO.DeleteTrackedItem(newItem)
				}
			} else {
				log.WithField("module", m.Key).Warnf(
					"unable to parse URL \"%s\" found in post \"%s\"",
					externalURL,
					webUrl,
				)
			}
		}
	}

	return nil
}

func (m *kemono) getExternalLinks(html string, comments *[]api.Comment) (links []string) {
	if !m.settings.ExternalURLs.DownloadExternalItems && !m.settings.ExternalURLs.PrintExternalItems {
		return links
	}

	document, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	document.Find("a").Each(func(index int, row *goquery.Selection) {
		uri, _ := row.Attr("href")
		if fileURL, parseErr := url.Parse(uri); parseErr == nil {
			if !strings.Contains(fileURL.String(), ".fanbox.cc/") {
				links = append(links, fileURL.String())
			}
		}
	})

	pattern := regexp.MustCompile(`(?m)https?://[-a-zA-Z0-9@:%._+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_+.~#?&/=!]*)`)
	urlMatches := pattern.FindAllStringSubmatch(html, -1)
	if len(urlMatches) > 0 {
		for _, match := range urlMatches {
			q, _ := url.Parse(match[0])
			q.Scheme = "https"

			if !strings.Contains(q.String(), ".fanbox.cc/") {
				links = append(links, q.String())
			}
		}
	}

	if comments != nil {
		for _, comment := range *comments {
			if comment.Content != "" {
				urlMatches = pattern.FindAllStringSubmatch(comment.Content, -1)
				if len(urlMatches) > 0 {
					for _, match := range urlMatches {
						q, _ := url.Parse(match[0])
						q.Scheme = "https"

						if !strings.Contains(q.String(), ".fanbox.cc/") {
							links = append(links, q.String())
						}
					}
				}
			}
		}
	}

	// remove archive links which got added for zip browsing while on the website
	var nonArchiveLinks []string
	for _, link := range links {
		if strings.HasPrefix(link, "/posts/archives/") {
			continue
		}
		nonArchiveLinks = append(nonArchiveLinks, link)
	}
	links = nonArchiveLinks

	// remove potential duplicates
	var uniqueLinks []string
	for _, link := range links {
		found := false
		for _, uniqueLink := range uniqueLinks {
			if uniqueLink == link {
				found = true
				break
			}
		}

		if !found {
			uniqueLinks = append(uniqueLinks, link)
		}
	}

	return uniqueLinks
}

func (m *kemono) getDownloadLinks(root *api.PostRoot) (links []models.DownloadQueueItem) {
	if root.Post.File.Path != "" {
		fileUri := fmt.Sprintf("%s/data/%s", m.baseUrl.String(), root.Post.File.Path)
		downloadItem := models.DownloadQueueItem{
			ItemID:  fileUri,
			FileURI: fileUri,
		}

		links = append(links, downloadItem)
	}

	for _, attachment := range root.Attachments {
		if attachment.Path != "" {
			fileUri := fmt.Sprintf("%s/data/%s", m.baseUrl.String(), attachment.Path)
			if attachment.Server != nil && *attachment.Server != "" {
				fileUri = fmt.Sprintf("%s/data/%s", *attachment.Server, attachment.Path)
			}

			downloadItem := models.DownloadQueueItem{
				ItemID:  fileUri,
				FileURI: fileUri,
			}

			links = append(links, downloadItem)
		}
	}

	// add thumbnails as fallback if we can't download the original file
	// or as additional download if we can't associate the thumbnail with an attachment/download
	for _, preview := range root.Previews {
		if preview.Path != "" {
			fileUri := fmt.Sprintf("%s/data/%s", m.baseUrl.String(), preview.Path)
			if preview.Server != nil && *preview.Server != "" {
				fileUri = fmt.Sprintf("%s/data/%s", *preview.Server, preview.Path)
			}

			// search for initial file name in attachments
			found := false
			for _, link := range links {
				if link.ItemID == fileUri {
					link.FallbackFileURI = fileUri
					found = true
					break
				}
			}

			if !found {
				downloadItem := models.DownloadQueueItem{
					ItemID:  fileUri,
					FileURI: fileUri,
				}
				links = append(links, downloadItem)
			}
		}
	}

	if len(root.Videos) > 0 {
		log.WithField("module", m.Key).Warn("found video download links, please implement video download handling")
		// exit early for now to see an actual use case of the video attribute instantly
		os.Exit(1)
	}

	return links
}
