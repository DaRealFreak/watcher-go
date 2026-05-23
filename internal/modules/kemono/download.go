package kemono

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"context"
	"github.com/DaRealFreak/watcher-go/internal/http/tls_session"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules"
	"github.com/DaRealFreak/watcher-go/internal/modules/kemono/api"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
	"github.com/DaRealFreak/watcher-go/pkg/linkfinder"
	"github.com/PuerkitoBio/goquery"
	"github.com/bogdanfinn/fhttp/http2"
	"log/slog"
)

// joinDataURL builds a "{host}/data/{path}" URL with exactly one slash between segments.
// API responses return paths with a leading slash (e.g. "/bd/86/...jpg"); naive
// fmt.Sprintf("%s/data/%s") would produce "host/data//bd/86/...jpg".
func joinDataURL(host, p string) string {
	return fmt.Sprintf("%s/data/%s", strings.TrimRight(host, "/"), strings.TrimLeft(p, "/"))
}

// extractDataPath pulls the hashed path out of any "{host}/data/{path}" URL.
// Returns "" if uri doesn't follow that shape.
func extractDataPath(uri string) string {
	if uri == "" {
		return ""
	}
	const marker = "/data/"
	idx := strings.Index(uri, marker)
	if idx < 0 {
		return ""
	}
	return uri[idx+len(marker):]
}

// isImageFile is a coarse "can this be served by /thumbnail/data/" check.
// img.kemono.cr only serves thumbnail-rendered images, not arbitrary content.
func isImageFile(name string) bool {
	switch strings.ToLower(path.Ext(name)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp":
		return true
	}
	return false
}

// thumbnailHost returns the img.<site> host used for /thumbnail/data/ URLs,
// picked from the current base URL (kemono.cr vs coomer.st).
func (m *kemono) thumbnailHost() string {
	if m.baseUrl != nil && strings.Contains(m.baseUrl.Host, "coomer") {
		return "img.coomer.st"
	}
	return "img.kemono.cr"
}

// buildThumbnailURL returns the img.<site>/thumbnail/data/<path> URL for an item,
// or "" if the file isn't an image or the path can't be derived.
func (m *kemono) buildThumbnailURL(item *models.DownloadQueueItem, fileName string) string {
	if !isImageFile(fileName) {
		// fall back to checking the URL path's own extension - the API-supplied
		// FileName isn't always set, in which case downloadPost derives fileName
		// from the URL which already has the hashed-path extension.
		if !isImageFile(item.FileURI) {
			return ""
		}
	}
	dataPath := extractDataPath(item.FallbackFileURI)
	if dataPath == "" {
		dataPath = extractDataPath(item.FileURI)
	}
	if dataPath == "" {
		return ""
	}
	return fmt.Sprintf("https://%s/thumbnail/data/%s", m.thumbnailHost(), strings.TrimLeft(dataPath, "/"))
}

// isNetworkError reports whether err looks like a transport-level failure
// (connect timeout, EOF mid-stream, connection reset) for which retrying via
// a fallback host is worthwhile.
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		return true
	}
	if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
		return true
	}
	// fhttp/tls-client wrap dial/connect errors as *url.Error; fall back to
	// substring matching since the underlying types are unexported.
	msg := err.Error()
	for _, needle := range []string{
		"connectex",                       // Windows winsock connect failures
		"connection refused",
		"connection reset",
		"no such host",
		"i/o timeout",
		"deadline exceeded",
		"network is unreachable",
		"host is unreachable",
		"TLS handshake timeout",
	} {
		if strings.Contains(msg, needle) {
			return true
		}
	}
	return false
}

func (m *kemono) processDownloadQueue(item *models.TrackedItem, downloadQueue []api.QuickPost, notifications ...*models.Notification) error {
	slog.Info(fmt.Sprintf("found %d new items for uri: \"%s\"", len(downloadQueue), item.URI), "module", m.Key)

	for _, notification := range notifications {
		slog.Log(context.Background(),
			notification.Level, notification.Message, "module", m.Key)
	}

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

func (m *kemono) downloadPost(item *models.TrackedItem, data api.QuickPost) error {
	webUrl := fmt.Sprintf("%s/%s/user/%s/post/%s", m.baseUrl.String(), data.Service, data.User, data.ID)
	post, err := m.api.GetPostDetails(data.Service, data.User, data.ID)
	if err != nil {
		return fmt.Errorf("failed to fetch post details: %w", err)
	}

	postComments, commentErr := m.api.GetPostComments(data.Service, data.User, data.ID)
	if commentErr != nil {
		return fmt.Errorf("failed to fetch post comments: %w", commentErr)
	}

	// set the download folder to the given post
	postFolderPath := fp.SanitizePath(data.ID, false)
	sanitizedPostTitle := fp.SanitizePath(data.Title, false)
	if strings.TrimSpace(sanitizedPostTitle) != "" {
		postFolderPath += " - " + sanitizedPostTitle
	}

	downloadLinks := m.getDownloadLinks(post)
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

		file := path.Join(
			m.GetDownloadDirectory(),
			m.Key,
			fp.TruncateMaxLength(m.getSubFolder(item)),
			fp.TruncateMaxLength(postFolderPath),
			fp.TruncateMaxLength(strings.TrimSpace(fmt.Sprintf("%s_%d_%s", data.ID, index+1, fileName))),
		)
		if err = m.Session.DownloadFile(file, downloadItem.FileURI); err != nil {
			var scErr http2.StreamError
			if errors.As(err, &scErr) {
				slog.Warn(fmt.Sprintf("received stream error \"%s\", trying to download in chunks", err.Error()), "module", m.Key)
				err = m.downloadChunks(
					downloadItem.FileURI,
					file,
					1*1024*1024,
					5,
					5*time.Second,
				)
			}

			// On 404 OR a transport-level network error against a CDN node, retry
			// the download through the main host (which 302s back to the origin's
			// preferred CDN node, possibly a different one).
			var statusErr tls_session.StatusError
			is404 := errors.As(err, &statusErr) && statusErr.StatusCode == 404
			isNetErr := isNetworkError(err)
			if (is404 || isNetErr) && downloadItem.FallbackFileURI != "" {
				reason := "404"
				if isNetErr && !is404 {
					reason = "network error"
				}
				slog.Warn(fmt.Sprintf("received %s \"%s\" downloading \"%s\", retrying via fallback url \"%s\"",
					reason,
					err.Error(),
					downloadItem.FileURI,
					downloadItem.FallbackFileURI), "module", m.Key)

				err = m.Session.DownloadFile(
					path.Join(
						m.GetDownloadDirectory(),
						m.Key,
						fp.TruncateMaxLength(m.getSubFolder(item)),
						fp.TruncateMaxLength(fp.SanitizePath(data.ID, false)),
						fp.TruncateMaxLength(strings.TrimSpace(fmt.Sprintf("%s_%d_fallback_%s", data.ID, index+1, fileName))),
					),
					downloadItem.FallbackFileURI,
				)
			}

			// Last-resort fallback for images: the dedicated img.<site>/thumbnail
			// host serves a downscaled JPEG rendered from the same content hash.
			// We use this only when both the CDN node and the origin redirect are
			// unreachable, since the result is a degraded version of the file.
			if err != nil {
				var s tls_session.StatusError
				stillRetryable := errors.As(err, &s) && s.StatusCode == 404
				stillRetryable = stillRetryable || isNetworkError(err)
				if stillRetryable {
					if thumbURL := m.buildThumbnailURL(downloadItem, fileName); thumbURL != "" {
						slog.Warn(fmt.Sprintf("primary and fallback downloads failed for \"%s\" (last error: %s), saving thumbnail from \"%s\" as degraded fallback",
							downloadItem.FileURI,
							err.Error(),
							thumbURL), "module", m.Key)

						if thumbErr := m.Session.DownloadFile(
							path.Join(
								m.GetDownloadDirectory(),
								m.Key,
								fp.TruncateMaxLength(m.getSubFolder(item)),
								fp.TruncateMaxLength(fp.SanitizePath(data.ID, false)),
								fp.TruncateMaxLength(strings.TrimSpace(fmt.Sprintf("%s_%d_thumbnail_%s", data.ID, index+1, fileName))),
							),
							thumbURL,
						); thumbErr == nil {
							// thumbnail saved - treat the item as recovered
							err = nil
						} else {
							slog.Warn(fmt.Sprintf("thumbnail fallback also failed for \"%s\": %s",
								thumbURL, thumbErr.Error()), "module", m.Key)
						}
					}
				}
			}

			if err != nil {
				return err
			}
		}
	}

	externalLinks := m.getExternalLinks(post, postComments)
	factory := modules.GetModuleFactory()
	for _, externalURL := range externalLinks {
		if m.settings.ExternalURLs.PrintExternalItems {
			slog.Info(fmt.Sprintf("found external URL: \"%s\" in post \"%s\"",
				externalURL,
				webUrl), "module", m.Key)
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
				if m.Cfg.Run.Force && newItem.CurrentItem != "" {
					slog.Info(fmt.Sprintf("resetting progress for item %s (current id: %s)", newItem.URI, newItem.CurrentItem), "module", m.Key)
					newItem.CurrentItem = ""
					m.DbIO.ChangeTrackedItemCompleteStatus(newItem, false)
					m.DbIO.UpdateTrackedItem(newItem, "")
				}

				if err = module.Parse(newItem); err != nil {
					slog.Warn(fmt.Sprintf("unable to parse external URL \"%s\" found in post \"%s\" with error \"%s\", skipping",
						newItem.URI,
						webUrl,
						err.Error()), "module", m.Key)
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
				slog.Warn(fmt.Sprintf("unable to parse URL \"%s\" found in post \"%s\"",
					externalURL,
					webUrl), "module", m.Key)
			}
		}
	}

	// try to create the download folder if we have external links but no direct download links
	if len(downloadLinks) == 0 && len(externalLinks) > 0 {
		downloadFolder := path.Join(
			m.GetDownloadDirectory(),
			m.Key,
			fp.TruncateMaxLength(m.getSubFolder(item)),
			fp.TruncateMaxLength(postFolderPath),
		)
		err = os.MkdirAll(downloadFolder, os.ModePerm)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *kemono) getExternalLinks(post *api.PostRoot, comments []api.Comment) (links []string) {
	if !m.settings.ExternalURLs.DownloadExternalItems && !m.settings.ExternalURLs.PrintExternalItems {
		return links
	}

	html := post.Post.Content
	if post.Post.Embed.Url != "" {
		links = append(links, post.Post.Embed.Url)
	}

	htmlLinks := linkfinder.GetLinks(html)
	for _, link := range htmlLinks {
		if !strings.Contains(link, ".fanbox.cc/") && !strings.Contains(link, "discord.gg/") {
			links = append(links, strings.Replace(link, "http://", "https://", 1))
		}
	}

	for _, comment := range comments {
		if comment.Commenter != post.Post.User {
			continue
		}

		if comment.Content != "" {
			commentLinks := linkfinder.GetLinks(comment.Content)
			for _, link := range commentLinks {
				if !strings.Contains(link, ".fanbox.cc/") && !strings.Contains(link, "discord.gg/") {
					links = append(links, strings.Replace(link, "http://", "https://", 1))
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

func (m *kemono) getDownloadLinks(root *api.PostRoot) (links []*models.DownloadQueueItem) {
	// Collect canonical attachments from post.file + post.attachments.
	// The API's top-level `attachments` array is empty in current responses;
	// the real list lives on post.attachments. previews[]/videos[] carry the
	// per-file CDN server hint that we apply below.
	type pendingFile struct {
		Name string
		Path string
	}
	pending := make([]pendingFile, 0)
	if root.Post.File.Path != "" {
		pending = append(pending, pendingFile{Name: root.Post.File.Name, Path: root.Post.File.Path})
	}
	for _, a := range root.Post.Attachments {
		if a.Path != "" {
			pending = append(pending, pendingFile{Name: a.Name, Path: a.Path})
		}
	}

	mainHost := m.baseUrl.String()
	for _, pf := range pending {
		mainURI := joinDataURL(mainHost, pf.Path)
		links = append(links, &models.DownloadQueueItem{
			ItemID:   mainURI,
			FileURI:  mainURI,
			FileName: pf.Name,
		})
	}

	// previews[] gives us the CDN host per path. Promote that to the primary
	// FileURI (faster, avoids origin redirect) and keep the main-host URL as a
	// fallback so we can recover from CDN-node connectivity issues.
	for _, preview := range root.Previews {
		// ignore mega folder icons
		if preview.Name == "https://mega.nz/rich-file.png" {
			continue
		}
		if preview.Path == "" {
			continue
		}

		mainURI := joinDataURL(mainHost, preview.Path)
		cdnURI := mainURI
		if preview.Server != nil && *preview.Server != "" {
			cdnURI = joinDataURL(*preview.Server, preview.Path)
		}

		matched := false
		for _, link := range links {
			if link.FileURI == mainURI {
				if cdnURI != mainURI {
					link.FileURI = cdnURI
					link.ItemID = cdnURI
					link.FallbackFileURI = mainURI
				}
				matched = true
				break
			}
		}
		if matched {
			continue
		}

		// unmatched preview - add as standalone download with the same fallback semantics
		item := &models.DownloadQueueItem{
			ItemID:   cdnURI,
			FileURI:  cdnURI,
			FileName: preview.Name,
		}
		if cdnURI != mainURI {
			item.FallbackFileURI = mainURI
		}
		links = append(links, item)
	}

	for _, video := range root.Videos {
		if video.Path == "" {
			continue
		}
		mainURI := joinDataURL(mainHost, video.Path)
		cdnURI := mainURI
		if video.Server != nil && *video.Server != "" {
			cdnURI = joinDataURL(*video.Server, video.Path)
		}

		matched := false
		for _, link := range links {
			if link.FileURI == cdnURI || link.FileURI == mainURI {
				if video.Name != "" {
					link.FileName = video.Name
				}
				if link.FileURI == mainURI && cdnURI != mainURI {
					link.FileURI = cdnURI
					link.ItemID = cdnURI
					link.FallbackFileURI = mainURI
				}
				matched = true
				break
			}
		}
		if matched {
			continue
		}

		item := &models.DownloadQueueItem{
			ItemID:   cdnURI,
			FileURI:  cdnURI,
			FileName: video.Name,
		}
		if cdnURI != mainURI {
			item.FallbackFileURI = mainURI
		}
		links = append(links, item)
	}

	document, _ := goquery.NewDocumentFromReader(strings.NewReader(root.Post.Content))
	document.Find("img").Each(func(index int, item *goquery.Selection) {
		src, exists := item.Attr("src")
		if exists {
			fileUri := src
			if !strings.HasPrefix(fileUri, "https") {
				fileUri = fmt.Sprintf("%s/%s", m.baseUrl.String(), src)
			}

			downloadItem := models.DownloadQueueItem{
				ItemID:  fileUri,
				FileURI: fileUri,
			}

			links = append(links, &downloadItem)
		}
	})

	document.Find("a[href^='/']").Each(func(index int, item *goquery.Selection) {
		if href, exists := item.Attr("href"); exists {
			fileUri := fmt.Sprintf("%s%s", m.baseUrl.String(), href)
			parsedLink, _ := url.Parse(href) // Ignore error; fileName defaults to empty if parsing fails
			fileName := ""

			if f := parsedLink.Query().Get("f"); f != "" {
				fileName = fp.SanitizePath(f, false)
				queryVals := parsedLink.Query()
				queryVals.Del("f")
				parsedLink.RawQuery = queryVals.Encode()
			}

			checkUrl := parsedLink.String()

			downloadItem := models.DownloadQueueItem{
				ItemID:   fileUri,
				FileURI:  fileUri,
				FileName: fileName,
			}

			added := false
			for _, link := range links {
				if strings.Contains(link.FileURI, checkUrl) {
					link.FallbackFileURI = fileUri
					added = true
					break
				}
			}

			if !added {
				links = append(links, &downloadItem)
			}
		}
	})

	return links
}
