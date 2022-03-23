package patreon

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"

	"github.com/DaRealFreak/watcher-go/internal/raven"
)

// userResponse contains not all but the relevant data of the user response of the public API
type userResponse struct {
	Data struct {
		Attributes struct {
			FullName string `json:"full_name"`
			Vanity   string `json:"vanity"`
		} `json:"attributes"`
		ID            json.Number `json:"id"`
		Relationships struct {
			Campaign struct {
				Data *struct {
					ID   json.Number `json:"id"`
					Type string      `json:"type"`
				} `json:"data"`
			} `json:"campaign"`
		} `json:"relationships"`
		Type string `json:"type"`
	} `json:"data"`
}

// getCreatorID extracts the creator ID from the campaign URI
func (m *patreon) getCreatorID(campaignURI string) (int, error) {
	res, err := m.Session.Get(campaignURI)
	if err != nil {
		return 0, err
	}

	creatorIDMatches := m.creatorIdPattern.FindStringSubmatch(m.Session.GetDocument(res).Text())
	if len(creatorIDMatches) != 2 {
		return 0, fmt.Errorf("unexpected amount of matches in search of creator id ")
	}

	creatorID, _ := strconv.ParseInt(creatorIDMatches[1], 10, 64)

	return int(creatorID), nil
}

// getCreatorCampaign returns the campaign data of the creator ID
func (m *patreon) getCreatorCampaign(creatorID int) (*userResponse, error) {
	res, err := m.Session.Get(fmt.Sprintf("https://www.patreon.com/api/user/%d", creatorID))
	if err != nil {
		return nil, err
	}

	reader, err := gzip.NewReader(res.Body)
	if err != nil {
		return nil, err
	}

	readerRes, readerErr := ioutil.ReadAll(reader)
	raven.CheckError(readerErr)

	var apiUserResponse userResponse
	if err := json.Unmarshal(readerRes, &apiUserResponse); err != nil {
		return nil, err
	}

	if apiUserResponse.Data.Relationships.Campaign.Data == nil {
		return nil, fmt.Errorf("user has no campaign")
	}

	return &apiUserResponse, nil
}
