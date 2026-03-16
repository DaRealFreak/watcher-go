package fantia

import (
	"fmt"
	"log/slog"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
)

var fanclubIDPattern = regexp.MustCompile(`/fanclubs/(\d+)`)

type mediaItem struct {
	fileName string
	fileURI  string
}

func (m *fantia) parseFanclub(item *models.TrackedItem) error {
	fanclubID := m.extractFanclubID(item.URI)

	slog.Info(
		fmt.Sprintf("parsing fanclub %s", fanclubID),
		"module", m.Key,
	)

	currentPostID := 0
	if item.CurrentItem != "" {
		currentPostID, _ = strconv.Atoi(item.CurrentItem)
	}

	// collect new post IDs from paginated HTML pages (newest first)
	var newPostIDs []string
	page := 1
	foundCurrentItem := false

	for {
		ids, err := m.getPostIDs(fanclubID, page)
		if err != nil {
			return err
		}

		if len(ids) == 0 {
			break
		}

		for _, id := range ids {
			num, _ := strconv.Atoi(id)
			if num <= currentPostID {
				foundCurrentItem = true
				continue
			}
			newPostIDs = append(newPostIDs, id)
		}

		if foundCurrentItem {
			break
		}
		page++
	}

	if len(newPostIDs) == 0 {
		return nil
	}

	// reverse to process oldest first
	for i, j := 0, len(newPostIDs)-1; i < j; i, j = i+1, j-1 {
		newPostIDs[i], newPostIDs[j] = newPostIDs[j], newPostIDs[i]
	}

	slog.Info(
		fmt.Sprintf("found %d new posts for uri: \"%s\"", len(newPostIDs), item.URI),
		"module", m.Key,
	)

	for i, postID := range newPostIDs {
		slog.Info(
			fmt.Sprintf(
				"downloading updates for uri: \"%s\" (%0.2f%%)",
				item.URI,
				float64(i+1)/float64(len(newPostIDs))*100,
			),
			"module", m.Key,
		)

		if err := m.downloadPost(postID, item); err != nil {
			slog.Warn(
				fmt.Sprintf("failed to process post %s, skipping: %s", postID, err.Error()),
				"module", m.Key,
			)
			continue
		}

		m.DbIO.UpdateTrackedItem(item, postID)
	}

	return nil
}

func (m *fantia) parsePost(item *models.TrackedItem) error {
	postID := m.extractPostID(item.URI)

	if err := m.downloadPost(postID, item); err != nil {
		return err
	}

	m.DbIO.ChangeTrackedItemCompleteStatus(item, true)
	return nil
}

func (m *fantia) downloadPost(postID string, trackedItem *models.TrackedItem) error {
	post, err := m.getPost(postID)
	if err != nil {
		return err
	}

	tag := fmt.Sprintf("%d", post.Fanclub.ID)
	if post.Fanclub.User.Name != "" {
		tag = post.Fanclub.User.Name
	}

	items := m.extractMediaFromPost(post)
	if len(items) == 0 {
		return nil
	}

	for _, mi := range items {
		filePath := path.Join(
			m.GetDownloadDirectory(),
			m.Key,
			fp.TruncateMaxLength(fp.SanitizePath(trackedItem.SubFolder, false)),
			fp.TruncateMaxLength(fp.SanitizePath(tag, false)),
			fp.TruncateMaxLength(fp.SanitizePath(mi.fileName, false)),
		)

		if err = m.Session.DownloadFile(filePath, mi.fileURI); err != nil {
			return err
		}
	}

	return nil
}

func (m *fantia) extractMediaFromPost(post *postData) []mediaItem {
	var items []mediaItem

	// add post thumbnail
	if post.Thumb != nil && post.Thumb.Original != "" {
		ext := extractExtensionFromURL(post.Thumb.Original)
		items = append(items, mediaItem{
			fileName: fmt.Sprintf("%d_thumb.%s", post.ID, ext),
			fileURI:  post.Thumb.Original,
		})
	}

	for _, content := range post.PostContents {
		if content.VisibleStatus != "visible" {
			planName := ""
			if content.Plan != nil {
				planName = fmt.Sprintf(" (plan: %s, %d JPY)", content.Plan.Name, content.Plan.Price)
			}
			slog.Warn(
				fmt.Sprintf("post %d content \"%s\" is locked%s, skipping", post.ID, content.Title, planName),
				"module", m.Key,
			)
			continue
		}

		switch content.Category {
		case "photo_gallery", "photo":
			for i, photo := range content.PostContentPhotos {
				photoURL := getBestPhotoURL(photo)
				if photoURL == "" {
					continue
				}
				ext := extractExtensionFromURL(photoURL)
				items = append(items, mediaItem{
					fileName: fmt.Sprintf("%d_%d_%d.%s", post.ID, content.ID, i+1, ext),
					fileURI:  photoURL,
				})
			}

		case "blog":
			blogURLs := extractBlogImages(content.Comment)
			for i, imgURL := range blogURLs {
				ext := extractExtensionFromURL(imgURL)
				items = append(items, mediaItem{
					fileName: fmt.Sprintf("%d_%d_blog_%d.%s", post.ID, content.ID, i+1, ext),
					fileURI:  imgURL,
				})
			}

		case "file", "download":
			if content.DownloadURI == "" {
				continue
			}
			downloadURL := content.DownloadURI
			if strings.HasPrefix(downloadURL, "/") {
				downloadURL = "https://fantia.jp" + downloadURL
			}
			fileName := content.Filename
			if fileName == "" {
				fileName = fmt.Sprintf("%d_%d_download", post.ID, content.ID)
			}
			items = append(items, mediaItem{
				fileName: fmt.Sprintf("%d_%s", post.ID, fileName),
				fileURI:  downloadURL,
			})
		}
	}

	return items
}

func (m *fantia) extractFanclubID(uri string) string {
	matches := fanclubIDPattern.FindStringSubmatch(uri)
	if len(matches) > 1 {
		return matches[1]
	}
	// try to extract from path directly
	uri = strings.TrimRight(uri, "/")
	parts := strings.Split(uri, "/")
	return parts[len(parts)-1]
}

func (m *fantia) extractPostID(uri string) string {
	matches := postIDPattern.FindStringSubmatch(uri)
	if len(matches) > 1 {
		return matches[1]
	}
	uri = strings.TrimRight(uri, "/")
	parts := strings.Split(uri, "/")
	return parts[len(parts)-1]
}

func extractExtensionFromURL(rawURL string) string {
	// strip query parameters
	if idx := strings.Index(rawURL, "?"); idx >= 0 {
		rawURL = rawURL[:idx]
	}
	if idx := strings.LastIndex(rawURL, "."); idx >= 0 {
		ext := rawURL[idx+1:]
		if len(ext) <= 5 {
			return ext
		}
	}
	return "jpg"
}
