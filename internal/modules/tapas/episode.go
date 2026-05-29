package tapas

import (
	"fmt"
	"html"
	"log/slog"
	"net/url"
	"path"
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules/tapas/api"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
	"github.com/PuerkitoBio/goquery"
)

// episodeAction is the policy decision for how to handle an episode based on
// its release/payment state.
type episodeAction int

const (
	// actionDownload: episode is downloadable.
	actionDownload episodeAction = iota
	// actionSkipAndAdvance: episode is not downloadable but won't become
	// downloadable later either (paid, unlocked-only content). Advance the
	// tracker so the warning isn't logged every run.
	actionSkipAndAdvance
	// actionStop: episode hasn't been released yet. Stop walking the list and
	// do not advance the tracker, so the next run picks the same id back up
	// once it goes live.
	actionStop
)

// episodeActionFor decides what to do with an episode based purely on its
// metadata. Pulled out as a pure function so the policy is unit-testable
// without a live API.
func episodeActionFor(ep api.Episode) episodeAction {
	if ep.Scheduled {
		return actionStop
	}
	if ep.MustPay && !ep.Unlocked {
		return actionSkipAndAdvance
	}
	return actionDownload
}

// parseEpisodeItem handles the case where the user tracks a single episode
// URL directly. Once downloaded the item is marked complete.
func (m *tapas) parseEpisodeItem(item *models.TrackedItem) error {
	match := m.episodeURI.FindStringSubmatch(item.URI)
	if len(match) < 2 {
		return fmt.Errorf("could not extract episode id from %q", item.URI)
	}

	advance, err := m.downloadEpisode(item, match[1])
	if err != nil {
		return err
	}
	if !advance {
		return nil
	}

	m.DbIO.UpdateTrackedItem(item, match[1])
	m.DbIO.ChangeTrackedItemCompleteStatus(item, true)

	return nil
}

// downloadEpisode fetches the episode payload and downloads every comic page
// belonging to it. The returned advance flag tells the caller whether the
// tracker should be moved forward past this episode: false means try again
// next run (e.g. the episode is scheduled for a future release).
func (m *tapas) downloadEpisode(item *models.TrackedItem, episodeID string) (advance bool, err error) {
	episode, err := m.api.Episode(episodeID)
	if err != nil {
		return false, err
	}

	switch episodeActionFor(episode.Episode) {
	case actionStop:
		when := strings.TrimSpace(episode.Episode.RelativePublishDate)
		if when == "" {
			when = "soon"
		}
		slog.Warn(
			fmt.Sprintf("episode %s is scheduled for a future release (%s), stopping", episodeID, when),
			"module", m.Key,
		)
		return false, nil
	case actionSkipAndAdvance:
		slog.Warn(
			fmt.Sprintf("episode %s requires payment and is not unlocked, skipping", episodeID),
			"module", m.Key,
		)
		return true, nil
	}

	imageURLs, err := extractEpisodeImages(episode.HTML)
	if err != nil {
		return false, err
	}

	if len(imageURLs) == 0 {
		slog.Warn(
			fmt.Sprintf("episode %s contained no downloadable images", episodeID),
			"module", m.Key,
		)
		return true, nil
	}

	episodeFolder := buildEpisodeFolder(episode.Episode)

	for index, imageURL := range imageURLs {
		fileName := buildPageFileName(index, imageURL)
		filePath := path.Join(
			m.GetDownloadDirectory(),
			m.Key,
			fp.TruncateMaxLength(fp.SanitizePath(item.SubFolder, false)),
			fp.TruncateMaxLength(fp.SanitizePath(episodeFolder, false)),
			fp.TruncateMaxLength(fp.SanitizePath(fileName, false)),
		)

		if err := m.Session.DownloadFile(filePath, imageURL); err != nil {
			return false, err
		}
	}

	return true, nil
}

// extractEpisodeImages scrapes the lazy-loaded image URLs out of the HTML
// fragment returned by the single-episode endpoint.
func extractEpisodeImages(htmlBody string) ([]string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlBody))
	if err != nil {
		return nil, err
	}

	var images []string
	doc.Find("img.content__img").Each(func(_ int, s *goquery.Selection) {
		src, ok := s.Attr("data-src")
		if !ok || src == "" {
			src, ok = s.Attr("src")
			if !ok || strings.HasPrefix(src, "data:") {
				return
			}
		}
		images = append(images, html.UnescapeString(src))
	})

	return images, nil
}

// buildEpisodeFolder returns the per-episode subfolder name. The numeric id is
// prefixed so episodes sort naturally on disk and remain unique when titles
// collide.
func buildEpisodeFolder(episode api.Episode) string {
	title := episode.Title
	if title == "" {
		title = fmt.Sprintf("episode_%s", episode.ID.String())
	}

	return fmt.Sprintf("%s_%s", episode.ID.String(), title)
}

// buildPageFileName returns a stable on-disk filename for a comic page based
// on its position in the episode. The CDN URL's own -N suffix cannot be used
// because tapas groups multi-image uploads and restarts the suffix at 0 in
// every group, which causes collisions when an episode mixes groups.
func buildPageFileName(index int, imageURL string) string {
	ext := ".png"
	if parsed, err := url.Parse(imageURL); err == nil {
		if e := path.Ext(path.Base(parsed.Path)); e != "" {
			ext = e
		}
	}

	return fmt.Sprintf("%03d%s", index+1, ext)
}
