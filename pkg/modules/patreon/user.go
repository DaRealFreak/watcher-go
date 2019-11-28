package patreon

import (
	"fmt"
	"strconv"
)

// getCreatorID extracts the creator ID from the campaign URI
func (m *patreon) getCreatorID(campaignURI string) (int, error) {
	res, err := m.Session.Get(campaignURI)
	if err != nil {
		return 0, err
	}

	creatorIDMatches := m.creatorIPattern.FindStringSubmatch(m.Session.GetDocument(res).Text())
	if len(creatorIDMatches) != 2 {
		return 0, fmt.Errorf("unexpected amount of matches in search of creator id ")
	}

	creatorID, _ := strconv.ParseInt(creatorIDMatches[1], 10, 64)

	return int(creatorID), nil
}
