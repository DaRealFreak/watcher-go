package pixiv

import (
	"encoding/json"
	"fmt"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/url"
)

// parse illustrations of artists
func (m *pixiv) parseUserIllustrations(item *models.TrackedItem) {
	userId := m.getUserIdFromUrl(item.Uri)
	if m.getUserDetail(userId) == nil {
		log.Info("couldn't retrieve user details, changing artist to complete")
		m.DbIO.ChangeTrackedItemCompleteStatus(item, true)
		return
	}

	var downloadQueue []models.DownloadQueueItem
	foundCurrentItem := false
	apiUrl := m.getUserIllustsUrl(userId, SearchFilterAll, 0)

	for !foundCurrentItem {
		fmt.Println(apiUrl)
		response := m.getUserIllusts(apiUrl)
		apiUrl = response.NextUrl
		for _, userIllustration := range response.Illustrations {
			if string(userIllustration.Id) == item.CurrentItem {
				foundCurrentItem = true
			}
		}

		// break if we don't have another page
		if apiUrl == "" {
			break
		}
	}

	fmt.Println(downloadQueue)
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

// build the user illustrations page URL manually
func (m *pixiv) getUserIllustsUrl(userId string, filter string, offset int) string {
	apiUrl, _ := url.Parse("https://app-api.pixiv.net/v1/user/illusts")
	data := url.Values{
		"user_id": {userId},
		"filter":  {"for_ios"},
	}

	// add passed options to the url values
	if filter != "" {
		data.Add("type", filter)
	}
	if offset > 0 {
		data.Add("offset", string(offset))
	}
	apiUrl.RawQuery = data.Encode()
	return apiUrl.String()
}

// retrieve user illustrations directly by url since the API response returns the next page url directly
func (m *pixiv) getUserIllusts(apiUrl string) *userWorkResponse {
	var userWorks userWorkResponse
	res, err := m.get(apiUrl)
	m.CheckError(err)

	response, err := ioutil.ReadAll(res.Body)
	m.CheckError(err)

	err = json.Unmarshal(response, &userWorks)
	m.CheckError(err)
	return &userWorks
}
