package pixiv

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"

	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/DaRealFreak/watcher-go/pkg/raven"
	log "github.com/sirupsen/logrus"
)

// parseUserIllustrations parses illustrations of artists
func (m *pixiv) parseUserIllustrations(item *models.TrackedItem) {
	userID := m.getUserIDFromURL(item.URI)
	if m.getUserDetail(userID) == nil {
		log.WithField("module", m.Key()).Warning(
			"couldn't retrieve user details, changing artist to complete",
		)
		m.DbIO.ChangeTrackedItemCompleteStatus(item, true)
		return
	}

	var downloadQueue []*downloadQueueItem
	foundCurrentItem := false
	apiURL := m.getUserIllustsURL(userID, SearchFilterAll, 0)

	for !foundCurrentItem {
		response := m.getUserIllusts(apiURL)
		apiURL = response.NextURL
		for _, userIllustration := range response.Illustrations {
			if string(userIllustration.ID) == item.CurrentItem {
				foundCurrentItem = true
				break
			}
			m.parseWork(userIllustration, &downloadQueue)
		}

		// break if we don't have another page
		if apiURL == "" {
			break
		}
	}

	// reverse download queue to download old items first
	for i, j := 0, len(downloadQueue)-1; i < j; i, j = i+1, j-1 {
		downloadQueue[i], downloadQueue[j] = downloadQueue[j], downloadQueue[i]
	}
	m.processDownloadQueue(downloadQueue, item)
}

// processDownloadQueue processes the download queue of user illustration download queue items
func (m *pixiv) processDownloadQueue(downloadQueue []*downloadQueueItem, trackedItem *models.TrackedItem) {
	log.WithField("module", m.Key()).Info(
		fmt.Sprintf("found %d new items for uri: %s", len(downloadQueue), trackedItem.URI),
	)

	for index, data := range downloadQueue {
		var err error
		log.WithField("module", m.Key()).Info(
			fmt.Sprintf(
				"downloading updates for uri: %s (%0.2f%%)",
				trackedItem.URI,
				float64(index+1)/float64(len(downloadQueue))*100,
			),
		)
		if data.Illustration.Type == SearchFilterIllustration || data.Illustration.Type == SearchFilterManga {
			err = m.downloadIllustration(data)
		} else if data.Illustration.Type == SearchFilterUgoira {
			err = m.downloadUgoira(data)
		}
		// ToDo: download novels as .txt

		raven.CheckError(err)
		m.DbIO.UpdateTrackedItem(trackedItem, data.ItemID)
	}
}

// getUserIDFromURL extracts the user ID from the passed URL
func (m *pixiv) getUserIDFromURL(uri string) string {
	u, _ := url.Parse(uri)
	q, _ := url.ParseQuery(u.RawQuery)
	if len(q["id"]) == 0 {
		raven.CheckError(fmt.Errorf("parsed uri(%s) does not contain any \"id\" tag", uri))
	}
	return q["id"][0]
}

// getUserDetail returns the user details from the API
func (m *pixiv) getUserDetail(userID string) *userDetailResponse {
	apiURL, _ := url.Parse("https://app-api.pixiv.net/v1/user/detail")
	data := url.Values{
		"user_id": {userID},
		"filter":  {"for_ios"},
	}
	apiURL.RawQuery = data.Encode()
	res, err := m.Session.Get(apiURL.String())
	raven.CheckError(err)

	// user got deleted or deactivated his account
	if res != nil && (res.StatusCode == 403 || res.StatusCode == 404) {
		return nil
	}

	response, err := ioutil.ReadAll(res.Body)
	raven.CheckError(err)

	var details userDetailResponse
	err = json.Unmarshal(response, &details)
	raven.CheckError(err)
	return &details
}

// getUserIllustsURL builds the user illustrations page URL manually
func (m *pixiv) getUserIllustsURL(userID string, filter string, offset int) string {
	apiURL, _ := url.Parse("https://app-api.pixiv.net/v1/user/illusts")
	data := url.Values{
		"user_id": {userID},
		"filter":  {"for_ios"},
	}

	// add passed options to the url values
	if filter != "" {
		data.Add("type", filter)
	}
	if offset > 0 {
		data.Add("offset", string(offset))
	}
	apiURL.RawQuery = data.Encode()
	return apiURL.String()
}

// getUserIllusts returns user illustrations directly by URL since the API response returns the next page URL directly
func (m *pixiv) getUserIllusts(apiURL string) *userWorkResponse {
	var userWorks userWorkResponse
	res, err := m.Session.Get(apiURL)
	raven.CheckError(err)

	response, err := ioutil.ReadAll(res.Body)
	raven.CheckError(err)

	raven.CheckError(json.Unmarshal(response, &userWorks))
	return &userWorks
}

// parseWork parses the different work types (illustration/manga/ugoira/novels) to append to the download queue
func (m *pixiv) parseWork(userIllustration *illustration, downloadQueue *[]*downloadQueueItem) {
	switch userIllustration.Type {
	case SearchFilterIllustration, SearchFilterManga:
		downloadQueueItem := downloadQueueItem{
			ItemID:       string(userIllustration.ID),
			DownloadTag:  fmt.Sprintf("%s/%s", userIllustration.User.ID, m.SanitizePath(userIllustration.User.Name, false)),
			Illustration: userIllustration,
		}
		*downloadQueue = append(*downloadQueue, &downloadQueueItem)
	case SearchFilterUgoira:
		// retrieve metadata later on download to prevent getting detected as harvesting software
		downloadQueueItem := downloadQueueItem{
			ItemID:       string(userIllustration.ID),
			DownloadTag:  fmt.Sprintf("%s/%s", userIllustration.User.ID, m.SanitizePath(userIllustration.User.Name, false)),
			Illustration: userIllustration,
		}
		*downloadQueue = append(*downloadQueue, &downloadQueueItem)
	case SearchFilterNovel:
		// ToDo: parse novel types
		fmt.Println(userIllustration)
	default:
		raven.CheckError(fmt.Errorf("unknown illustration type: %s", userIllustration.Type))
	}
}

// getUgoiraFrame returns the corresponding frame for the passed file name from the passed ugoira metadata
func (m *pixiv) getUgoiraFrame(fileName string, metadata *ugoiraMetadata) (*frame, error) {
	for _, frame := range metadata.Frames {
		if frame.File == fileName {
			return frame, nil
		}
	}
	return nil, fmt.Errorf("no frame found for file: %s", fileName)
}
