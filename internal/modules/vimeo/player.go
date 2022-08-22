package vimeo

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/DaRealFreak/watcher-go/internal/models"
)

type PlayerJson struct {
	Request struct {
		Files struct {
			Dash struct {
				CDNs struct {
					AkfireInterconnectQuic struct {
						Url string `json:"url"`
					} `json:"akfire_interconnect_quic"`
				} `json:"cdns"`
			} `json:"dash"`
			HLS struct {
				CDNs struct {
					AkfireInterconnectQuic struct {
						Url string `json:"url"`
					} `json:"akfire_interconnect_quic"`
				} `json:"cdns"`
			} `json:"hls"`
		} `json:"files"`
	} `json:"request"`
	Video struct {
		ID    json.Number `json:"id"`
		Title string      `json:"title"`
		Owner struct {
			Name string `json:"name"`
		} `json:"owner"`
	} `json:"video"`
}

func (m *vimeo) getPlayerJSON(item *models.TrackedItem) (*PlayerJson, error) {
	results := m.defaultVideoURLPattern.FindStringSubmatch(item.URI)
	if len(results) < 2 || len(results) > 3 {
		return nil, fmt.Errorf("unsupported URL format")
	}

	playerUrl := fmt.Sprintf("https://player.vimeo.com/video/%s/config", results[1])
	if len(results) == 3 && results[2] != "" {
		playerUrl += "?h=" + results[2]
	}

	res, err := m.Session.Get(playerUrl)
	if err != nil {
		return nil, err
	}

	out, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		return nil, readErr
	}

	var playerJson PlayerJson
	err = json.Unmarshal(out, &playerJson)

	return &playerJson, err
}

func (p *PlayerJson) GetMasterJSONUrl() string {
	// HLS is for apple products while MPEG-DASH is the international standard, so use dash by default
	if p.Request.Files.Dash.CDNs.AkfireInterconnectQuic.Url != "" {
		return p.Request.Files.Dash.CDNs.AkfireInterconnectQuic.Url
	} else {
		return p.Request.Files.HLS.CDNs.AkfireInterconnectQuic.Url
	}
}

func (p *PlayerJson) GetVideoTitle() string {
	return p.Video.Title
}
