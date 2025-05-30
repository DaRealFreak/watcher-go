package pixivapi

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPixivAPI_tokenRenewal(t *testing.T) {
	pixivAPI := getTestPixivAPI()

	err := pixivAPI.ConfigureTokenSource()
	assert.New(t).NoError(err)

	apiURL, _ := url.Parse("https://app-api.pixiv.net/v1/illust/detail")
	data := url.Values{
		"illust_id": {"123456"},
	}
	apiURL.RawQuery = data.Encode()

	res, err := pixivAPI.Session.Get(apiURL.String())
	assert.New(t).NoError(err)
	assert.New(t).Equal(200, res.StatusCode)

	// expire token to make the Valid() function of the token to return false and trigger a token renewal
	pixivAPI.token.Expiry = time.Now().Add(-1 * time.Minute)

	res, err = pixivAPI.Session.Get(apiURL.String())
	assert.New(t).NoError(err)
	assert.New(t).Equal(200, res.StatusCode)
}
