package coomerfans

import (
	"context"
	"fmt"
	"log/slog"
	"path"
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
	"github.com/PuerkitoBio/goquery"
)

// extractPostMedia returns the content media URLs of a post page: images and
// video sources inside `div.post-body`. Avatar/recommended thumbnails live
// outside that container and are excluded. URLs are returned in document
// order (images before video sources), de-duplicated.
func extractPostMedia(doc *goquery.Document) []string {
	var urls []string
	seen := make(map[string]bool)
	add := func(u string) {
		if u != "" && !seen[u] {
			seen[u] = true
			urls = append(urls, u)
		}
	}
	doc.Find("div.post-body img[src]").Each(func(_ int, s *goquery.Selection) {
		src, _ := s.Attr("src")
		add(src)
	})
	doc.Find("div.post-body video source[src]").Each(func(_ int, s *goquery.Selection) {
		src, _ := s.Attr("src")
		add(src)
	})
	return urls
}

// scrapeUsername returns the creator username from the first `/u/` link on a
// post page, or "" if none is present. Used to derive the download folder for
// single-post tracked items (which carry no creator subfolder).
func scrapeUsername(doc *goquery.Document) string {
	var username string
	doc.Find("a[href^='/u/']").EachWithBreak(func(_ int, s *goquery.Selection) bool {
		href, ok := s.Attr("href")
		if !ok {
			return true
		}
		if _, _, name, ok := parseUserURL(href); ok {
			username = name
			return false
		}
		return true
	})
	return username
}

// processDownloadQueue downloads each queued post in order, advancing the
// tracked item's progress after each post completes.
func (m *coomerfans) processDownloadQueue(item *models.TrackedItem, queue []postRef, notifications ...*models.Notification) error {
	slog.Info(fmt.Sprintf("found %d new items for uri: %q", len(queue), item.URI), "module", m.Key)

	for _, notification := range notifications {
		slog.Log(context.Background(), notification.Level, notification.Message, "module", m.Key)
	}

	for index, ref := range queue {
		slog.Info(fmt.Sprintf(
			"downloading updates for uri: %q (%0.2f%%)",
			item.URI,
			float64(index+1)/float64(len(queue))*100,
		), "module", m.Key)

		if err := m.downloadPost(item, ref); err != nil {
			return err
		}

		m.DbIO.UpdateTrackedItem(item, ref.ID)
	}

	return nil
}

// downloadPost fetches a post page, extracts its media, and downloads each file
// to {downloadDir}/coomerfans.com/{service}/{username}/{postId} - {title}/{file}.
func (m *coomerfans) downloadPost(item *models.TrackedItem, ref postRef) error {
	postURL := fmt.Sprintf("%s/p/%s/%s/%s", baseURL, ref.ID, ref.UserID, ref.Service)
	resp, err := m.Session.Get(postURL)
	if err != nil {
		return fmt.Errorf("failed to fetch post %s: %w", ref.ID, err)
	}

	doc := m.Session.GetDocument(resp)
	mediaURLs := extractPostMedia(doc)
	if len(mediaURLs) == 0 {
		return nil
	}

	subFolder := m.getSubFolder(item)
	if subFolder == "" {
		if username := scrapeUsername(doc); username != "" {
			subFolder = fp.SanitizePath(fmt.Sprintf("%s/%s", ref.Service, username), true)
		}
	}

	postFolder := fp.SanitizePath(ref.ID, false)
	if title := fp.SanitizePath(ref.Title, false); strings.TrimSpace(title) != "" {
		postFolder += " - " + title
	}

	for index, mediaURL := range mediaURLs {
		fileName := fp.SanitizePath(fp.GetFileName(mediaURL), false)
		target := path.Join(
			m.GetDownloadDirectory(),
			m.Key,
			fp.TruncateMaxLength(fp.SanitizePath(subFolder, false)),
			fp.TruncateMaxLength(postFolder),
			fp.TruncateMaxLength(strings.TrimSpace(fmt.Sprintf("%s_%d_%s", ref.ID, index+1, fileName))),
		)
		if err := m.Session.DownloadFile(target, mediaURL); err != nil {
			return err
		}
	}

	return nil
}
