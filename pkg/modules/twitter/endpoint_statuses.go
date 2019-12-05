package twitter

import (
	"encoding/json"
	"net/url"
)

// Tweet contains the struct to unmarshal twitters API responses
type Tweet struct {
	CreatedAt string      `json:"created_at"`
	ID        json.Number `json:"id"`
	Text      string      `json:"text"`
	Entities  struct {
		MediaElement []*struct {
			ID            json.Number `json:"id"`
			MediaURLHTTPS string      `json:"media_url_https"`
			DisplayURL    string      `json:"display_url"`
			Type          string      `json:"type"`
		} `json:"media"`
	} `json:"entities"`
	RetweetedStatus interface{} `json:"retweeted_status"`
}

func (m *twitter) getUserTimeline(values url.Values) (apiRes []*Tweet, apiErr *APIError, err error) {
	apiURI := "https://api.twitter.com/1.1/statuses/user_timeline.json"
	if values.Encode() != "" {
		apiURI += "?" + values.Encode()
	}

	res, err := m.Session.Get(apiURI)
	if err != nil {
		return nil, nil, err
	}

	// map the http.Response into either the api response or the api error
	err = m.mapAPIResponse(res, &apiRes, &apiErr)

	return apiRes, apiErr, err
}
