package tapas

import (
	"fmt"
	"log/slog"
	"strconv"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules/tapas/api"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
)

// parseSeries walks the series episode list from oldest to newest and
// processes every episode published after the current tracked position.
func (m *tapas) parseSeries(item *models.TrackedItem) error {
	seriesID, err := m.resolveSeriesID(item.URI)
	if err != nil {
		return err
	}

	if item.SubFolder == "" {
		title, err := m.api.SeriesTitle(seriesID)
		if err != nil {
			return err
		}

		subFolder := fp.SanitizePath(title, false)
		m.DbIO.ChangeTrackedItemSubFolder(item, subFolder)
		item.SubFolder = subFolder
	}

	newEpisodes, err := m.collectNewEpisodes(seriesID, item.CurrentItem)
	if err != nil {
		return err
	}

	if len(newEpisodes) == 0 {
		return nil
	}

	slog.Info(
		fmt.Sprintf("found %d new episodes for series \"%s\"", len(newEpisodes), item.URI),
		"module", m.Key,
	)

	for index, episode := range newEpisodes {
		slog.Info(
			fmt.Sprintf(
				"downloading episode %s (%s) for series \"%s\" (%0.2f%%)",
				episode.ID,
				episode.Title,
				item.URI,
				float64(index+1)/float64(len(newEpisodes))*100,
			),
			"module", m.Key,
		)

		advance, err := m.downloadEpisode(item, episode.ID)
		if err != nil {
			return err
		}
		if !advance {
			return nil
		}

		m.DbIO.UpdateTrackedItem(item, episode.ID)
	}

	return nil
}

// collectNewEpisodes returns the ordered list of episodes published after the
// currently tracked episode id. The list is oldest-first so that interrupting
// the download leaves the tracker at a valid intermediate position.
func (m *tapas) collectNewEpisodes(seriesID, currentItemID string) ([]api.EpisodeListItem, error) {
	currentID, _ := strconv.Atoi(currentItemID)

	var result []api.EpisodeListItem

	page := 1
	for {
		items, pagination, err := m.api.EpisodeList(seriesID, page)
		if err != nil {
			return nil, err
		}

		for _, episode := range items {
			id, convErr := strconv.Atoi(episode.ID)
			if convErr != nil {
				continue
			}
			if id <= currentID {
				continue
			}
			result = append(result, episode)
		}

		if !pagination.HasNext {
			break
		}
		page++
	}

	return result, nil
}
