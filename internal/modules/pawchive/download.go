package pawchive

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"strings"

	"log/slog"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules"
	"github.com/DaRealFreak/watcher-go/internal/modules/pawchive/api"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
	"github.com/DaRealFreak/watcher-go/pkg/linkfinder"
	"github.com/PuerkitoBio/goquery"
)

// buildFileURL returns the file-host download URL for a hashed path. The optional
// ?f={name} sets the served filename to match what the website uses.
func (m *pawchive) buildFileURL(path, name string) string {
	u := fmt.Sprintf("%s/data/%s", fileHost, strings.TrimLeft(path, "/"))
	if name != "" {
		u = fmt.Sprintf("%s?f=%s", u, url.QueryEscape(name))
	}
	return u
}

// extractDataPath pulls the hashed path out of any ".../data/<path>" URL,
// dropping any "?f=" query pawchive appends. Returns "" if uri has no /data/.
func extractDataPath(uri string) string {
	if uri == "" {
		return ""
	}
	const marker = "/data/"
	idx := strings.Index(uri, marker)
	if idx < 0 {
		return ""
	}
	p := uri[idx+len(marker):]
	if q := strings.IndexByte(p, '?'); q >= 0 {
		p = p[:q]
	}
	return p
}

// isImageFile is a coarse "can img.pawchive.st render a thumbnail for this" check;
// the thumbnail host only serves rendered images, not arbitrary content.
func isImageFile(name string) bool {
	switch strings.ToLower(path.Ext(name)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp":
		return true
	}
	return false
}

// buildThumbnailURL returns the img.pawchive.st/thumbnail/data/<path> URL for an
// image download, or "" if the file isn't an image or its path can't be derived.
func (m *pawchive) buildThumbnailURL(item *models.DownloadQueueItem, fileName string) string {
	dataPath := extractDataPath(item.FileURI)
	if dataPath == "" {
		return ""
	}
	// prefer the served filename, falling back to the hashed path's own extension
	// (downloadPost doesn't always have a FileName for inline images).
	if !isImageFile(fileName) && !isImageFile(dataPath) {
		return ""
	}
	return fmt.Sprintf("%s/thumbnail/data/%s", imageHost, strings.TrimLeft(dataPath, "/"))
}

// fileDownloadTarget decides how to fetch a single file of a post, returning the
// URL to download, whether that URL is a downscaled thumbnail, and whether the
// file should be skipped entirely.
//
// When a post's full-res files are not archived yet (has_full=false), the file
// host doesn't reliably 404 - it often returns 504 (gateway timeout), which would
// fatal the whole parse. Since the API already tells us the full file is missing,
// we never request it: images fall back to the img.pawchive.st thumbnail and
// non-image files (e.g. .rar) are skipped, mirroring the site's "Missing N
// full-res photos, M files" state. Archived posts, and files not hosted on
// pawchive's own file host (external inline images, which has_full does not
// govern), are downloaded directly.
func (m *pawchive) fileDownloadTarget(post *api.Post, item *models.DownloadQueueItem, fileName string) (downloadURL string, isThumbnail, skip bool) {
	if post.HasFull || extractDataPath(item.FileURI) == "" {
		return item.FileURI, false, false
	}

	thumbURL := m.buildThumbnailURL(item, fileName)
	if thumbURL == "" {
		return "", false, true
	}
	return thumbURL, true, false
}

// getDownloadLinks collects the post's own files: post.file (if present), each
// attachment, and any inline <img> in the rendered content.
func (m *pawchive) getDownloadLinks(post *api.Post) (links []*models.DownloadQueueItem) {
	type pendingFile struct {
		Name string
		Path string
	}
	pending := make([]pendingFile, 0)
	if post.File.Path != "" {
		pending = append(pending, pendingFile{Name: post.File.Name, Path: post.File.Path})
	}
	for _, a := range post.Attachments {
		if a.Path != "" {
			pending = append(pending, pendingFile{Name: a.Name, Path: a.Path})
		}
	}

	for _, pf := range pending {
		// ignore mega folder icons
		if pf.Name == "https://mega.nz/rich-file.png" {
			continue
		}
		fileURI := m.buildFileURL(pf.Path, pf.Name)
		links = append(links, &models.DownloadQueueItem{
			ItemID:   fileURI,
			FileURI:  fileURI,
			FileName: pf.Name,
		})
	}

	document, _ := goquery.NewDocumentFromReader(strings.NewReader(post.Content))
	document.Find("img").Each(func(_ int, sel *goquery.Selection) {
		src, exists := sel.Attr("src")
		if !exists {
			return
		}
		fileURI := src
		if !strings.HasPrefix(fileURI, "http://") && !strings.HasPrefix(fileURI, "https://") {
			fileURI = fmt.Sprintf("%s/%s", m.baseUrl.String(), strings.TrimLeft(src, "/"))
		}
		links = append(links, &models.DownloadQueueItem{
			ItemID:  fileURI,
			FileURI: fileURI,
		})
	})

	return links
}

// getExternalLinks extracts external download URLs (mega/gdrive/etc.) from the post
// content, its embed, and comments authored by the creator. Gated by settings.
func (m *pawchive) getExternalLinks(post *api.Post, comments []api.Comment) (links []string) {
	if !m.settings.ExternalURLs.DownloadExternalItems && !m.settings.ExternalURLs.PrintExternalItems {
		return links
	}

	if post.Embed.Url != "" {
		links = append(links, post.Embed.Url)
	}

	for _, link := range linkfinder.GetLinks(post.Content) {
		if !strings.Contains(link, ".fanbox.cc/") && !strings.Contains(link, "discord.gg/") {
			links = append(links, strings.Replace(link, "http://", "https://", 1))
		}
	}

	for _, comment := range comments {
		if comment.Commenter != post.User || comment.Content == "" {
			continue
		}
		for _, link := range linkfinder.GetLinks(comment.Content) {
			if !strings.Contains(link, ".fanbox.cc/") && !strings.Contains(link, "discord.gg/") {
				links = append(links, strings.Replace(link, "http://", "https://", 1))
			}
		}
	}

	// remove duplicates, preserving order
	var uniqueLinks []string
	for _, link := range links {
		found := false
		for _, ul := range uniqueLinks {
			if ul == link {
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

func (m *pawchive) processDownloadQueue(item *models.TrackedItem, downloadQueue []api.Post) error {
	slog.Info(fmt.Sprintf("found %d new items for uri: \"%s\"", len(downloadQueue), item.URI), "module", m.Key)

	for index, data := range downloadQueue {
		slog.Info(fmt.Sprintf(
			"downloading updates for uri: \"%s\" (%0.2f%%)",
			item.URI,
			float64(index+1)/float64(len(downloadQueue))*100,
		), "module", m.Key)

		if err := m.downloadPost(item, data); err != nil {
			return err
		}

		m.DbIO.UpdateTrackedItem(item, data.ID)
	}

	return nil
}

func (m *pawchive) downloadPost(item *models.TrackedItem, post api.Post) error {
	webUrl := fmt.Sprintf("%s/%s/user/%s/post/%s", m.baseUrl.String(), post.Service, post.User, post.ID)

	postComments, commentErr := m.api.GetPostComments(post.Service, post.User, post.ID)
	if commentErr != nil {
		return fmt.Errorf("failed to fetch post comments: %w", commentErr)
	}

	postFolderPath := fp.SanitizePath(post.ID, false)
	sanitizedTitle := fp.SanitizePath(post.Title, false)
	if strings.TrimSpace(sanitizedTitle) != "" {
		postFolderPath += " - " + sanitizedTitle
	}

	downloadLinks := m.getDownloadLinks(&post)
	for index, downloadItem := range downloadLinks {
		parsedLink, parsedErr := url.Parse(downloadItem.FileURI)
		if parsedErr != nil {
			return parsedErr
		}

		fileName := fp.GetFileName(downloadItem.FileURI)
		if f := parsedLink.Query().Get("f"); f != "" {
			fileName = f
		}
		if downloadItem.FileName != "" {
			fileName = downloadItem.FileName
		}
		fileName = fp.SanitizePath(fileName, false)

		// ignore mega folder icons
		if strings.Contains(fileName, "mega.nz_rich-folder") {
			continue
		}

		downloadURL, isThumbnail, skip := m.fileDownloadTarget(&post, downloadItem, fileName)
		if skip {
			slog.Warn(fmt.Sprintf(
				"post \"%s\" is not archived yet (has_full=false) and file \"%s\" has no thumbnail fallback, skipping file",
				webUrl, downloadItem.FileURI), "module", m.Key)
			continue
		}

		namePart := fileName
		if isThumbnail {
			slog.Warn(fmt.Sprintf(
				"post \"%s\" is not archived yet (has_full=false); saving thumbnail \"%s\" as degraded fallback for full file \"%s\"",
				webUrl, downloadURL, downloadItem.FileURI), "module", m.Key)
			namePart = "thumbnail_" + fileName
		}

		file := path.Join(
			m.GetDownloadDirectory(),
			m.Key,
			fp.TruncateMaxLength(m.getSubFolder(item)),
			fp.TruncateMaxLength(postFolderPath),
			fp.TruncateMaxLength(strings.TrimSpace(fmt.Sprintf("%s_%d_%s", post.ID, index+1, namePart))),
		)
		if err := m.Session.DownloadFile(file, downloadURL); err != nil {
			// A thumbnail is already a degraded fallback; if even that fails, warn
			// and move on rather than fataling the parse. A failed full-res download
			// on an archived post is a genuine error and still propagates.
			if isThumbnail {
				slog.Warn(fmt.Sprintf(
					"thumbnail fallback failed for \"%s\": %s, skipping file",
					downloadURL, err.Error()), "module", m.Key)
				continue
			}
			return err
		}
	}

	externalLinks := m.getExternalLinks(&post, postComments)
	factory := modules.GetModuleFactory()
	for _, externalURL := range externalLinks {
		if m.settings.ExternalURLs.PrintExternalItems {
			slog.Info(fmt.Sprintf("found external URL: \"%s\" in post \"%s\"", externalURL, webUrl), "module", m.Key)
		}

		if m.settings.ExternalURLs.DownloadExternalItems {
			if factory.CanParse(externalURL) {
				module := factory.GetModuleFromURI(externalURL)
				if err := module.Load(); err != nil {
					return err
				}
				newItem := m.DbIO.GetFirstOrCreateTrackedItem(externalURL, "", module)
				// don't delete previously already added items
				deleteAfter := newItem.CurrentItem == ""
				if m.Cfg.Run.Force && newItem.CurrentItem != "" {
					slog.Info(fmt.Sprintf("resetting progress for item %s (current id: %s)", newItem.URI, newItem.CurrentItem), "module", m.Key)
					newItem.CurrentItem = ""
					m.DbIO.ChangeTrackedItemCompleteStatus(newItem, false)
					m.DbIO.UpdateTrackedItem(newItem, "")
				}

				if err := module.Parse(newItem); err != nil {
					slog.Warn(fmt.Sprintf("unable to parse external URL \"%s\" found in post \"%s\" with error \"%s\", skipping",
						newItem.URI, webUrl, err.Error()), "module", m.Key)
					if !m.settings.ExternalURLs.SkipErrorsForExternalURLs {
						if deleteAfter {
							m.DbIO.DeleteTrackedItem(newItem)
						}
						return err
					}
				}

				if deleteAfter {
					m.DbIO.DeleteTrackedItem(newItem)
				}
			} else {
				slog.Warn(fmt.Sprintf("unable to parse URL \"%s\" found in post \"%s\"", externalURL, webUrl), "module", m.Key)
			}
		}
	}

	// create the post folder if we only found external links (no direct downloads)
	if len(downloadLinks) == 0 && len(externalLinks) > 0 {
		downloadFolder := path.Join(
			m.GetDownloadDirectory(),
			m.Key,
			fp.TruncateMaxLength(m.getSubFolder(item)),
			fp.TruncateMaxLength(postFolderPath),
		)
		if err := os.MkdirAll(downloadFolder, os.ModePerm); err != nil {
			return err
		}
	}

	return nil
}
