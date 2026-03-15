package bsky

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"path"
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
)

type mediaPost struct {
	rkey  string
	tag   string
	items []mediaItem
}

type mediaItem struct {
	fileName string
	fileURI  string
	isVideo  bool
}

func (m *bsky) parseProfile(item *models.TrackedItem) error {
	handle := m.extractHandle(item.URI)

	profile, err := m.getProfile(handle)
	if err != nil {
		return err
	}

	slog.Info(
		fmt.Sprintf("parsing profile \"%s\" (%s)", profile.DisplayName, profile.Handle),
		"module", m.Key,
	)

	// resolve PDS for video blob downloads
	pdsURL := m.settings.PDS
	if resolved, resolveErr := m.resolvePDS(profile.DID); resolveErr == nil {
		pdsURL = resolved
	}

	// collect media posts (API returns newest first)
	var mediaPosts []mediaPost
	cursor := ""
	foundCurrentItem := false
	tag := profile.Handle

	for {
		feed, feedErr := m.getAuthorFeed(profile.DID, cursor)
		if feedErr != nil {
			return feedErr
		}

		for _, fi := range feed.Feed {
			// skip reposts (reason is set for reposts)
			if len(fi.Reason) > 0 && string(fi.Reason) != "null" {
				continue
			}

			rkey := m.extractRkey(fi.Post.URI)

			// skip already processed posts
			if item.CurrentItem != "" && rkey <= item.CurrentItem {
				foundCurrentItem = true
				continue
			}

			items := m.extractMediaFromPost(fi.Post, profile.DID, pdsURL)
			if len(items) > 0 {
				mediaPosts = append(mediaPosts, mediaPost{
					rkey:  rkey,
					tag:   tag,
					items: items,
				})
			}
		}

		if foundCurrentItem || feed.Cursor == "" || len(feed.Feed) == 0 {
			break
		}
		cursor = feed.Cursor
	}

	if len(mediaPosts) == 0 {
		return nil
	}

	// reverse to process oldest first
	for i, j := 0, len(mediaPosts)-1; i < j; i, j = i+1, j-1 {
		mediaPosts[i], mediaPosts[j] = mediaPosts[j], mediaPosts[i]
	}

	return m.processMediaPosts(mediaPosts, item)
}

func (m *bsky) processMediaPosts(mediaPosts []mediaPost, trackedItem *models.TrackedItem) error {
	total := len(mediaPosts)

	slog.Info(
		fmt.Sprintf("found %d new items for uri: \"%s\"", total, trackedItem.URI),
		"module", m.Key,
	)

	for i, mp := range mediaPosts {
		slog.Info(
			fmt.Sprintf(
				"downloading updates for uri: \"%s\" (%0.2f%%)",
				trackedItem.URI,
				float64(i+1)/float64(total)*100,
			),
			"module", m.Key,
		)

		for _, item := range mp.items {
			filePath := path.Join(
				m.GetDownloadDirectory(),
				m.Key,
				fp.TruncateMaxLength(fp.SanitizePath(trackedItem.SubFolder, false)),
				fp.TruncateMaxLength(fp.SanitizePath(mp.tag, false)),
				fp.TruncateMaxLength(fp.SanitizePath(item.fileName, false)),
			)

			if err := m.Session.DownloadFile(filePath, item.fileURI); err != nil {
				if item.isVideo {
					slog.Warn(
						fmt.Sprintf("failed to download video %s, skipping: %s", item.fileName, err.Error()),
						"module", m.Key,
					)
					continue
				}
				return err
			}
		}

		m.DbIO.UpdateTrackedItem(trackedItem, mp.rkey)
	}

	return nil
}

func (m *bsky) extractHandle(uri string) string {
	uri = strings.TrimRight(uri, "/")
	parts := strings.Split(uri, "/")
	for i, part := range parts {
		if part == "profile" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return uri
}

func (m *bsky) extractRkey(atURI string) string {
	// AT URI format: at://did:plc:xxx/app.bsky.feed.post/rkey
	parts := strings.Split(atURI, "/")
	return parts[len(parts)-1]
}

func (m *bsky) extractMediaFromPost(post postView, did string, pdsURL string) []mediaItem {
	if post.Embed == nil {
		return nil
	}

	var et embedType
	if err := json.Unmarshal(post.Embed, &et); err != nil {
		return nil
	}

	rkey := m.extractRkey(post.URI)

	switch et.Type {
	case "app.bsky.embed.images#view":
		return m.extractImages(post.Embed, rkey)
	case "app.bsky.embed.video#view":
		return m.extractVideo(post.Embed, rkey, did, pdsURL)
	case "app.bsky.embed.recordWithMedia#view":
		var rwm recordWithMediaEmbedView
		if err := json.Unmarshal(post.Embed, &rwm); err != nil {
			return nil
		}
		return m.extractMediaFromEmbed(rwm.Media, rkey, did, pdsURL)
	default:
		return nil
	}
}

func (m *bsky) extractMediaFromEmbed(embed json.RawMessage, rkey string, did string, pdsURL string) []mediaItem {
	if embed == nil {
		return nil
	}

	var et embedType
	if err := json.Unmarshal(embed, &et); err != nil {
		return nil
	}

	switch et.Type {
	case "app.bsky.embed.images#view":
		return m.extractImages(embed, rkey)
	case "app.bsky.embed.video#view":
		return m.extractVideo(embed, rkey, did, pdsURL)
	default:
		return nil
	}
}

func (m *bsky) extractImages(embedJSON json.RawMessage, rkey string) []mediaItem {
	var embed imagesEmbedView
	if err := json.Unmarshal(embedJSON, &embed); err != nil {
		return nil
	}

	var items []mediaItem
	for i, img := range embed.Images {
		ext := m.extractCDNExtension(img.Fullsize)
		items = append(items, mediaItem{
			fileName: fmt.Sprintf("%s_%d.%s", rkey, i+1, ext),
			fileURI:  img.Fullsize,
		})
	}

	return items
}

func (m *bsky) extractVideo(embedJSON json.RawMessage, rkey string, did string, pdsURL string) []mediaItem {
	var embed videoEmbedView
	if err := json.Unmarshal(embedJSON, &embed); err != nil {
		return nil
	}

	// use PDS blob endpoint to get the original uploaded video
	videoURL := fmt.Sprintf("%s/xrpc/com.atproto.sync.getBlob?did=%s&cid=%s",
		pdsURL, url.QueryEscape(did), url.QueryEscape(embed.CID))

	return []mediaItem{
		{
			fileName: fmt.Sprintf("%s_video.mp4", rkey),
			fileURI:  videoURL,
			isVideo:  true,
		},
	}
}

func (m *bsky) extractCDNExtension(cdnURL string) string {
	// CDN URL format: https://cdn.bsky.app/img/feed_fullsize/plain/did:plc:xxx/bafkreixxx@jpeg
	parsed, err := url.Parse(cdnURL)
	if err != nil {
		return "jpg"
	}

	base := path.Base(parsed.Path)
	if idx := strings.LastIndex(base, "@"); idx >= 0 {
		return base[idx+1:]
	}

	return "jpg"
}
