package deviantart

import (
	"encoding/json"
	"io/ioutil"
	"net/url"
)

// Placebo implements the API endpoint https://www.deviantart.com/api/v1/oauth2/placebo
func (m *deviantArt) Placebo() (response *PlaceboResponse, err error) {
	var placebo PlaceboResponse
	values := url.Values{
		"access_token": {m.token.AccessToken},
	}
	res, err := m.Session.Post("https://www.deviantart.com/api/v1/oauth2/placebo", values)
	if err != nil {
		return nil, err
	}
	content, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(content, &placebo); err != nil {
		return nil, err
	}
	return &placebo, nil
}
