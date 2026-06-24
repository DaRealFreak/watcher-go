package pawchive

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules/pawchive/api"
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
