package pixiv

import (
	"strconv"

	"github.com/DaRealFreak/watcher-go/internal/models"
)

func (m *pixiv) parseIllustration(item *models.TrackedItem) error {
	illustrationID, _ := strconv.ParseInt(m.patterns.illustrationPattern.FindStringSubmatch(item.URI)[1], 10, 64)

	details, err := m.mobileAPI.GetIllustDetail(int(illustrationID))
	if err != nil {
		return err
	}

	currentDownloadQueueItem := &downloadQueueItem{
		ItemID:       details.Illustration.ID,
		DownloadTag:  details.Illustration.User.GetUserTag(),
		DownloadItem: details.Illustration,
	}

	return m.processDownloadQueue([]*downloadQueueItem{currentDownloadQueueItem}, item)
}
