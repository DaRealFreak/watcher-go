package deviantart

import (
	"fmt"
	"strconv"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules/deviantart/napi"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
)

func (m *deviantArt) parseArtNapi(item *models.TrackedItem) error {
	results := m.daPattern.artPattern.FindStringSubmatch(item.URI)

	if len(results) != 3 {
		return fmt.Errorf("unexpected amount of matches in search deviantart ID")
	}

	deviationId, _ := strconv.ParseInt(results[2], 10, 64)
	deviation, err := m.nAPI.ExtendedDeviation(int(deviationId), results[1], napi.DeviationTypeArt, false, nil)
	if err != nil {
		return err
	}

	if item.SubFolder == "" {
		m.DbIO.ChangeTrackedItemSubFolder(item, deviation.Deviation.Author.Username)
	}

	dl := []downloadQueueItemNAPI{
		{
			itemID:      deviation.Deviation.DeviationId.String(),
			deviation:   deviation.Deviation,
			downloadTag: fp.SanitizePath(item.SubFolder, false),
		},
	}

	if err = m.processDownloadQueueNapi(dl, item); err != nil {
		return err
	}

	m.DbIO.ChangeTrackedItemCompleteStatus(item, true)

	return nil
}
