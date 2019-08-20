package pixiv

import (
	"encoding/json"
	"io/ioutil"
	"net/url"

	"github.com/DaRealFreak/watcher-go/pkg/raven"
)

// getUgoiraMetaData retrieves meta data of ugoira illustrations
func (m *pixiv) getUgoiraMetaData(illustrationID string) *ugoiraResponse {
	var ugoiraMetadataResponse ugoiraResponse
	apiURL, _ := url.Parse("https://app-api.pixiv.net/v1/ugoira/metadata")
	data := url.Values{
		"illust_id": {illustrationID},
	}
	apiURL.RawQuery = data.Encode()
	res, err := m.Session.Get(apiURL.String())
	raven.CheckError(err)

	response, err := ioutil.ReadAll(res.Body)
	raven.CheckError(err)

	err = json.Unmarshal(response, &ugoiraMetadataResponse)
	raven.CheckError(err)
	return &ugoiraMetadataResponse
}
