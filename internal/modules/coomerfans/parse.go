// Package coomerfans contains the implementation of the coomerfans.com module
package coomerfans

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
	"github.com/PuerkitoBio/goquery"
)

// postURLPattern matches a post URL: /p/{postId}/{userId}/{service}
var postURLPattern = regexp.MustCompile(`/p/(\d+)/(\d+)/(\w+)`)

// userURLPattern matches a creator URL: /u/{service}/{userId}/{username}
var userURLPattern = regexp.MustCompile(`/u/(\w+)/(\d+)/([^/?&]+)`)

// parsePostURL extracts the post ID, user ID and service from a post URL.
func parsePostURL(uri string) (postID, userID, service string, ok bool) {
	m := postURLPattern.FindStringSubmatch(uri)
	if len(m) != 4 {
		return "", "", "", false
	}
	return m[1], m[2], m[3], true
}

// parseUserURL extracts the service, user ID and username from a creator URL.
func parseUserURL(uri string) (service, userID, username string, ok bool) {
	m := userURLPattern.FindStringSubmatch(uri)
	if len(m) != 4 {
		return "", "", "", false
	}
	return m[1], m[2], m[3], true
}

// postRef identifies a single post discovered while scraping.
type postRef struct {
	ID      string
	UserID  string
	Service string
	Title   string
}

// extractPostRefs returns the posts listed on a creator page, in document
// order (newest-first). Each post is taken from its `div.post > h3 > a` link;
// the duplicate "View Post" link and avatar/recommended links are ignored.
func extractPostRefs(doc *goquery.Document) []postRef {
	var refs []postRef
	seen := make(map[string]bool)
	doc.Find("div.post").Each(func(_ int, sel *goquery.Selection) {
		a := sel.Find("h3 a[href^='/p/']").First()
		href, ok := a.Attr("href")
		if !ok {
			return
		}
		id, userID, service, ok := parsePostURL(href)
		if !ok || seen[id] {
			return
		}
		seen[id] = true
		refs = append(refs, postRef{
			ID:      id,
			UserID:  userID,
			Service: service,
			Title:   strings.TrimSpace(a.Text()),
		})
	})
	return refs
}

// subFolderForURI derives the "{service}/{username}" subfolder from a creator
// URL. Returns "" for any other URL shape (e.g. a direct post URL).
func subFolderForURI(uri string) string {
	service, _, username, ok := parseUserURL(uri)
	if !ok {
		return ""
	}
	return fp.SanitizePath(fmt.Sprintf("%s/%s", service, username), true)
}

// getSubFolder returns the item's stored subfolder, or derives one from the
// creator URL when none is set.
func (m *coomerfans) getSubFolder(item *models.TrackedItem) string {
	if item.SubFolder != "" {
		return item.SubFolder
	}
	return subFolderForURI(item.URI)
}

// parseUser pages through a creator's posts (newest-first), collecting every
// post newer than the last-seen one, then downloads them oldest-first.
func (m *coomerfans) parseUser(item *models.TrackedItem) error {
	service, userID, username, ok := parseUserURL(item.URI)
	if !ok {
		return fmt.Errorf("could not extract service/user from URL: %s", item.URI)
	}

	if item.SubFolder == "" {
		m.DbIO.ChangeTrackedItemSubFolder(item, m.getSubFolder(item))
	}

	var queue []postRef
	foundCurrent := false
	for page := 1; ; page++ {
		pageURL := fmt.Sprintf("%s/u/%s/%s/%s?page=%d", baseURL, service, userID, username, page)
		resp, err := m.Session.Get(pageURL)
		if err != nil {
			return fmt.Errorf("failed to fetch creator page %d: %w", page, err)
		}

		refs := extractPostRefs(m.Session.GetDocument(resp))
		if len(refs) == 0 {
			break
		}

		for _, ref := range refs {
			if ref.ID == item.CurrentItem {
				foundCurrent = true
				break
			}
			queue = append(queue, ref)
		}
		if foundCurrent {
			break
		}
	}

	// reverse to oldest-first so an interrupted run resumes cleanly
	for i, j := 0, len(queue)-1; i < j; i, j = i+1, j-1 {
		queue[i], queue[j] = queue[j], queue[i]
	}

	slog.Info(fmt.Sprintf("found user %s (%s/%s) with %d new posts", username, service, userID, len(queue)), "module", m.Key)

	return m.processDownloadQueue(item, queue)
}

// parsePost downloads a single post added directly as a tracked item.
func (m *coomerfans) parsePost(item *models.TrackedItem) error {
	id, userID, service, ok := parsePostURL(item.URI)
	if !ok {
		return fmt.Errorf("could not extract post ID from URL: %s", item.URI)
	}
	return m.processDownloadQueue(item, []postRef{{ID: id, UserID: userID, Service: service}})
}
