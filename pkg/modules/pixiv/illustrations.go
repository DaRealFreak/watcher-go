package pixiv

import (
	"encoding/json"
	"io/ioutil"
	"net/url"
)

// getUgoiraMetaData retrieves meta data of ugoira illustrations
func (m *pixiv) getUgoiraMetaData(illustrationID string) (apiRes *ugoiraResponse, err error) {
	var ugoiraMetadataResponse ugoiraResponse
	apiURL, _ := url.Parse("https://app-api.pixiv.net/v1/ugoira/metadata")
	data := url.Values{
		"illust_id": {illustrationID},
	}
	apiURL.RawQuery = data.Encode()
	res, err := m.Session.Get(apiURL.String())
	if err != nil {
		return nil, err
	}

	response, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(response, &ugoiraMetadataResponse)
	return &ugoiraMetadataResponse, err
}
