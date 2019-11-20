package pixiv

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"strconv"

	"github.com/DaRealFreak/watcher-go/pkg/models"
	"github.com/DaRealFreak/watcher-go/pkg/modules/pixiv/session"
	log "github.com/sirupsen/logrus"
)

// parseUserIllustrations parses illustrations of artists
func (m *pixiv) parseUserIllustrations(item *models.TrackedItem) (err error) {
	userID, err := m.getUserIDFromURL(item.URI)
	if err != nil {
		return err
	}

	userDetails, err := m.getUserDetail(userID)
	if err != nil {
		return err
	}

	if userDetails == nil {
		log.WithField("module", m.Key).Warning(
			"couldn't retrieve user details, changing artist to complete",
		)
		m.DbIO.ChangeTrackedItemCompleteStatus(item, true)

		return nil
	}

	var downloadQueue []*downloadQueueItem

	foundCurrentItem := false
	apiURL := m.getUserIllustsURL(userID, SearchFilterAll, 0)

	for !foundCurrentItem {
		response, err := m.getUserIllusts(apiURL)
		if err != nil {
			return err
		}

		apiURL = response.NextURL

		for _, userIllustration := range response.Illustrations {
			if string(userIllustration.ID) == item.CurrentItem {
				foundCurrentItem = true
				break
			}

			err = m.parseWork(userIllustration, &downloadQueue)
			if err != nil {
				return err
			}
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

	return m.processDownloadQueue(downloadQueue, item)
}

// processDownloadQueue processes the download queue of user illustration download queue items
func (m *pixiv) processDownloadQueue(downloadQueue []*downloadQueueItem, trackedItem *models.TrackedItem) (err error) {
	log.WithField("module", m.Key).Info(
		fmt.Sprintf("found %d new items for uri: %s", len(downloadQueue), trackedItem.URI),
	)

	for index, data := range downloadQueue {
		log.WithField("module", m.Key).Info(
			fmt.Sprintf(
				"downloading updates for uri: %s (%0.2f%%)",
				trackedItem.URI,
				float64(index+1)/float64(len(downloadQueue))*100,
			),
		)

		switch data.Illustration.Type {
		case SearchFilterIllustration, SearchFilterManga:
			err = m.downloadIllustration(data)
		case SearchFilterUgoira:
			err = m.downloadUgoira(data)
		}
		// ToDo: download novels as .txt

		if err != nil {
			switch err.(type) {
			case *session.FileNotFoundError:
				log.WithField("module", m.Key).Warningf(
					"404 status code received for ID %s, skipping item",
					data.ItemID,
				)
			default:
				return err
			}
		}

		m.DbIO.UpdateTrackedItem(trackedItem, data.ItemID)
	}

	return nil
}

// getUserIDFromURL extracts the user ID from the passed URL
func (m *pixiv) getUserIDFromURL(uri string) (string, error) {
	u, _ := url.Parse(uri)
	q, _ := url.ParseQuery(u.RawQuery)

	if len(q["id"]) == 0 {
		return "", fmt.Errorf("parsed uri(%s) does not contain any \"id\" tag", uri)
	}

	return q["id"][0], nil
}

// getUserDetail returns the user details from the API
func (m *pixiv) getUserDetail(userID string) (apiRes *userDetailResponse, err error) {
	apiURL, _ := url.Parse("https://app-api.pixiv.net/v1/user/detail")
	data := url.Values{
		"user_id": {userID},
		"filter":  {"for_ios"},
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

	var details userDetailResponse

	err = json.Unmarshal(response, &details)

	return &details, err
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
		data.Add("offset", strconv.Itoa(offset))
	}

	apiURL.RawQuery = data.Encode()

	return apiURL.String()
}

// getUserIllusts returns user illustrations directly by URL since the API response returns the next page URL directly
func (m *pixiv) getUserIllusts(apiURL string) (apiRes *userWorkResponse, err error) {
	var userWorks userWorkResponse

	res, err := m.Session.Get(apiURL)
	if err != nil {
		return nil, err
	}

	response, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(response, &userWorks)

	return &userWorks, err
}

// parseWork parses the different work types (illustration/manga/ugoira/novels) to append to the download queue
func (m *pixiv) parseWork(userIllustration *illustration, downloadQueue *[]*downloadQueueItem) error {
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
		return fmt.Errorf("unknown illustration type: %s", userIllustration.Type)
	}

	return nil
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
