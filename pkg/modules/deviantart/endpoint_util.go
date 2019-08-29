package deviantart

import (
	"encoding/json"
	"io/ioutil"
	"net/url"

	"github.com/DaRealFreak/watcher-go/pkg/raven"
)

// ScopeUtil is the required scope for the OAuth2 Token
const ScopeUtil = "basic"

// Placebo implements the API endpoint https://www.deviantart.com/api/v1/oauth2/placebo
func (m *deviantArt) Placebo() (response *UtilPlaceboResponse) {
	var placebo UtilPlaceboResponse
	values := url.Values{}
	res, err := m.deviantArtSession.APIPost("https://www.deviantart.com/api/v1/oauth2/placebo", values, ScopeUtil)
	raven.CheckError(err)

	content, err := ioutil.ReadAll(res.Body)
	raven.CheckError(err)

	// unmarshal the request content into the UtilPlaceboResponse
	raven.CheckError(json.Unmarshal(content, &placebo))
	return &placebo
}
