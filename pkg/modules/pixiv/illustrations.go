package pixiv

import (
	"encoding/json"
	"io/ioutil"
	"net/url"
)

// retrieve meta data of ugoira illustrations
func (m *pixiv) getUgoiraMetaData(illustrationId string) *ugoiraResponse {
	var ugoiraMetadataResponse ugoiraResponse
	apiUrl, _ := url.Parse("https://app-api.pixiv.net/v1/ugoira/metadata")
	data := url.Values{
		"illust_id": {illustrationId},
	}
	apiUrl.RawQuery = data.Encode()
	res, err := m.Session.Get(apiUrl.String())
	m.CheckError(err)

	response, err := ioutil.ReadAll(res.Body)
	m.CheckError(err)

	err = json.Unmarshal(response, &ugoiraMetadataResponse)
	m.CheckError(err)
	return &ugoiraMetadataResponse
}
