package pixiv

import (
	"fmt"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	"strconv"
)

func (m *pixiv) parseFanbox(item *models.TrackedItem) error {
	creatorID, _ := strconv.ParseInt(m.patterns.fanboxPattern.FindStringSubmatch(item.URI)[1], 10, 64)

	creatorInfo, err := m.ajaxAPI.GetPostList(int(creatorID), 200)
	if err != nil {
		return err
	}

	fmt.Println(creatorInfo)

	return nil
}
