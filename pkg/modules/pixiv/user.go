package pixiv

import (
	"encoding/json"
	"fmt"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	"io/ioutil"
	"log"
	"net/url"
)

// parse the artists
func (m *pixiv) parseUserIllustrations(item *models.TrackedItem) {
	artistId := m.getUserIdFromUrl(item.Uri)
	m.getUserDetail(artistId)
	fmt.Println(artistId)
}

func (m *pixiv) getUserIdFromUrl(uri string) string {
	u, _ := url.Parse(uri)
	q, _ := url.ParseQuery(u.RawQuery)
	if len(q["id"]) == 0 {
		log.Fatalf("parsed uri(%s) does not contain any \"id\" tag", uri)
	}
	return q["id"][0]
}

func (m *pixiv) getUserDetail(userId string) *userDetailResponse {
	apiUrl, _ := url.Parse("https://app-api.pixiv.net/v1/user/detail")
	data := url.Values{
		"user_id": {userId},
		"filter":  {"for_ios"},
	}
	apiUrl.RawQuery = data.Encode()
	res, err := m.get(apiUrl.String())
	m.CheckError(err)

	response, err := ioutil.ReadAll(res.Body)
	m.CheckError(err)

	var details userDetailResponse
	err = json.Unmarshal(response, &details)
	m.CheckError(err)
	return &details
}
