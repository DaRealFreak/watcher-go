package pixiv

import (
	"strconv"

	"github.com/DaRealFreak/watcher-go/pkg/models"
)

func (m *pixiv) parseIllustration(item *models.TrackedItem) error {
	illustrationID, _ := strconv.ParseInt(m.patterns.illustrationPattern.FindStringSubmatch(item.URI)[1], 10, 64)

	details, err := m.mobileAPI.GetIllustDetail(int(illustrationID))
	if err != nil {
		return err
	}

	downloadQueueItem := &downloadQueueItem{
		ItemID:       details.Illustration.ID,
		DownloadTag:  details.Illustration.User.GetUserTag(),
		DownloadItem: details.Illustration,
	}

	if details.Illustration.Type == Ugoira {
		return m.downloadUgoira(downloadQueueItem, details.Illustration.ID)
	}

	return m.downloadIllustration(downloadQueueItem, details.Illustration)
}
