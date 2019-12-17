package pixiv

import (
	"testing"

	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/stretchr/testify/assert"
)

func TestPixiv_Parse(t *testing.T) {
	pixiv := NewBareModule()

	assert.New(t).NoError(pixiv.Parse(&models.TrackedItem{
		URI: "https://www.pixiv.net/search.php?word=test&order=date_d",
	}))
	assert.New(t).NoError(pixiv.Parse(&models.TrackedItem{
		URI: "https://www.pixiv.net/en/tags/test/artworks",
	}))
	assert.New(t).NoError(pixiv.Parse(&models.TrackedItem{
		URI: "https://www.pixiv.net/fanbox/creator/420928",
	}))
	assert.New(t).NoError(pixiv.Parse(&models.TrackedItem{
		URI: "https://www.pixiv.net/fanbox/creator/420928/post",
	}))
	assert.New(t).NoError(pixiv.Parse(&models.TrackedItem{
		URI: "https://www.pixiv.net/member.php?id=420928",
	}))
	assert.New(t).NoError(pixiv.Parse(&models.TrackedItem{
		URI: "https://www.pixiv.net/member_illust.php?id=420928&type=illust",
	}))
	assert.New(t).NoError(pixiv.Parse(&models.TrackedItem{
		URI: "https://www.pixiv.net/en/artworks/75338094",
	}))
	assert.New(t).NoError(pixiv.Parse(&models.TrackedItem{
		URI: "https://www.pixiv.net/member_illust.php?mode=medium&illust_id=75338094",
	}))
}
