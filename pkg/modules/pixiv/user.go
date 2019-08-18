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

	var downloadQueue []*downloadQueueItem
	foundCurrentItem := false
	apiUrl := m.getUserIllustsUrl(userId, SearchFilterAll, 0)

	for !foundCurrentItem {
		response := m.getUserIllusts(apiUrl)
		apiUrl = response.NextUrl
		for _, userIllustration := range response.Illustrations {
			if string(userIllustration.Id) == item.CurrentItem {
				foundCurrentItem = true
				break
			}
			m.parseWork(userIllustration, &downloadQueue)
		}

		// break if we don't have another page
		if apiUrl == "" {
			break
		}
	}

	// reverse download queue to download old items first
	for i, j := 0, len(downloadQueue)-1; i < j; i, j = i+1, j-1 {
		downloadQueue[i], downloadQueue[j] = downloadQueue[j], downloadQueue[i]
	}
	m.processDownloadQueue(downloadQueue, item)
}

func (m *pixiv) processDownloadQueue(downloadQueue []*downloadQueueItem, trackedItem *models.TrackedItem) {
	log.Info(fmt.Sprintf("found %d new items for uri: %s", len(downloadQueue), trackedItem.Uri))

	for index, data := range downloadQueue {
		var err error
		log.Info(fmt.Sprintf("downloading updates for uri: %s (%0.2f%%)", trackedItem.Uri, float64(index+1)/float64(len(downloadQueue))*100))
		if data.Type == SearchFilterIllustration || data.Type == SearchFilterManga {
			err = m.downloadIllustration(data)
		} else if data.Type == SearchFilterUgoira {
			err = m.downloadUgoira(data)
		}
		// ToDo: download novels as .txt

		m.CheckError(err)
		m.DbIO.UpdateTrackedItem(trackedItem, data.ItemId)
	}
}

// extract the user ID from the passed url
func (m *pixiv) getUserIdFromUrl(uri string) string {
	u, _ := url.Parse(uri)
	q, _ := url.ParseQuery(u.RawQuery)
	if len(q["id"]) == 0 {
		log.Fatalf("parsed uri(%s) does not contain any \"id\" tag", uri)
	}
	return q["id"][0]
}

// retrieve the user details from the API
func (m *pixiv) getUserDetail(userId string) *userDetailResponse {
	apiUrl, _ := url.Parse("https://app-api.pixiv.net/v1/user/detail")
	data := url.Values{
		"user_id": {userId},
		"filter":  {"for_ios"},
	}
	apiUrl.RawQuery = data.Encode()
	res, err := m.Session.Get(apiUrl.String())
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
	res, err := m.Session.Get(apiUrl)
	m.CheckError(err)

	response, err := ioutil.ReadAll(res.Body)
	m.CheckError(err)

	err = json.Unmarshal(response, &userWorks)
	m.CheckError(err)
	return &userWorks
}

// differentiate the work types (illustration/manga/ugoira/novels)
func (m *pixiv) parseWork(userIllustration *illustration, downloadQueue *[]*downloadQueueItem) {
	if userIllustration.Type == SearchFilterIllustration || userIllustration.Type == SearchFilterManga {
		m.addMetaPages(userIllustration, downloadQueue)
	} else if userIllustration.Type == SearchFilterUgoira {
		m.addUgoiraWork(userIllustration, downloadQueue)
	} else if userIllustration.Type == SearchFilterNovel {
		// ToDo: parse novel types
		return
	} else {
		log.Fatal("unknown illustration type: " + userIllustration.Type)
	}
}

// add illustration/manga images to the passed download queue
func (m *pixiv) addMetaPages(userIllustration *illustration, downloadQueue *[]*downloadQueueItem) {
	for _, image := range userIllustration.MetaPages {
		downloadQueueItem := downloadQueueItem{
			ItemId:      string(userIllustration.Id),
			DownloadTag: fmt.Sprintf("%s/%s", userIllustration.User.Id, m.SanitizePath(userIllustration.User.Name, false)),
			FileName:    m.GetFileName(image["image_urls"]["original"]),
			FileUri:     image["image_urls"]["original"],
			Type:        userIllustration.Type,
		}
		*downloadQueue = append(*downloadQueue, &downloadQueueItem)
	}
	if len(userIllustration.MetaSinglePage) > 0 {
		downloadQueueItem := downloadQueueItem{
			ItemId:      string(userIllustration.Id),
			DownloadTag: fmt.Sprintf("%s/%s", userIllustration.User.Id, m.SanitizePath(userIllustration.User.Name, false)),
			FileName:    m.GetFileName(userIllustration.MetaSinglePage["original_image_url"]),
			FileUri:     userIllustration.MetaSinglePage["original_image_url"],
			Type:        userIllustration.Type,
		}
		*downloadQueue = append(*downloadQueue, &downloadQueueItem)
	}
}

// add ugoira works to the passed download queue
func (m *pixiv) addUgoiraWork(userIllustration *illustration, downloadQueue *[]*downloadQueueItem) {
	// retrieve metadata later on download to prevent getting detected as harvesting software
	downloadQueueItem := downloadQueueItem{
		ItemId:      string(userIllustration.Id),
		DownloadTag: fmt.Sprintf("%s/%s", userIllustration.User.Id, m.SanitizePath(userIllustration.User.Name, false)),
		FileName:    "",
		FileUri:     "",
		Type:        userIllustration.Type,
	}
	*downloadQueue = append(*downloadQueue, &downloadQueueItem)
}

// retrieve corresponding frame for the passed file name from the ugoira metadata
func (m *pixiv) getUgoiraFrame(fileName string, metadata *ugoiraMetadata) (*frame, error) {
	for _, frame := range metadata.Frames {
		if frame.File == fileName {
			return frame, nil
		}
	}
	return nil, fmt.Errorf("no frame found for file: %s", fileName)
}
