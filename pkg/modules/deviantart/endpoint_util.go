package deviantart

import (
	"encoding/json"
	"io/ioutil"
	"net/url"

	"github.com/DaRealFreak/watcher-go/pkg/raven"
)

// Placebo implements the API endpoint https://www.deviantart.com/api/v1/oauth2/placebo
func (m *deviantArt) Placebo() (response *PlaceboResponse) {
	var placebo PlaceboResponse
	values := url.Values{}
	res, err := m.deviantArtSession.APIPost("https://www.deviantart.com/api/v1/oauth2/placebo", values)
	raven.CheckError(err)

	content, err := ioutil.ReadAll(res.Body)
	raven.CheckError(err)

	// unmarshal the request content into the PlaceboResponse
	raven.CheckError(json.Unmarshal(content, &placebo))
	return &placebo
}
