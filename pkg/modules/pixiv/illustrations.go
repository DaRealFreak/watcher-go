package pixiv

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"regexp"

	"github.com/DaRealFreak/watcher-go/pkg/models"
)

// parseUserIllustration parses a single illustration
func (m *pixiv) parseUserIllustration(item *models.TrackedItem) (err error) {
	illustID, err := m.getIllustIDFromURL(item.URI)
	if err != nil {
		return err
	}

	apiRes, err := m.getIllustDetail(illustID)
	if err != nil {
		return err
	}

	var downloadQueue []*downloadQueueItem
	if err := m.parseWork(apiRes.Illustration, &downloadQueue); err != nil {
		return err
	}

	return m.processDownloadQueue(downloadQueue, item)
}

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

// getIllustDetail returns the illustration details from the API
func (m *pixiv) getIllustDetail(illustID string) (apiRes *illustrationDetailResponse, err error) {
	apiURL, _ := url.Parse("https://app-api.pixiv.net/v1/illust/detail")
	data := url.Values{
		"illust_id": {illustID},
	}
	apiURL.RawQuery = data.Encode()
	res, err := m.Session.Get(apiURL.String())

	if err != nil {
		return nil, err
	}

	// user got deleted or deactivated his account
	if res != nil && (res.StatusCode == 403 || res.StatusCode == 404) {
		return nil, nil
	}

	response, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var details illustrationDetailResponse
	err = json.Unmarshal(response, &details)

	return &details, err
}

// getIllustIDFromURL extracts the illustration ID from the passed URL
func (m *pixiv) getIllustIDFromURL(uri string) (string, error) {
	u, _ := url.Parse(uri)
	prettyURLPattern := regexp.MustCompile(`/artworks/(?P<ID>\d+)(?:/|$|\?)`)

	if prettyURLPattern.MatchString(u.Path) {
		return prettyURLPattern.FindStringSubmatch(u.Path)[1], nil
	}

	q, _ := url.ParseQuery(u.RawQuery)
	if len(q["illust_id"]) == 0 {
		return "", fmt.Errorf("parsed uri(%s) does not contain any \"illust_id\" tag", uri)
	}

	return q["illust_id"][0], nil
}
