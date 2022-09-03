package vimeo

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/DaRealFreak/watcher-go/internal/models"
)

type PlayerJson struct {
	Request struct {
		Files struct {
			Dash struct {
				CDNs struct {
					AkfireInterconnectQuic *CDN `json:"akfire_interconnect_quic"`
					FastlySkyfire          *CDN `json:"fastly_skyfire"`
				} `json:"cdns"`
			} `json:"dash"`
			HLS struct {
				CDNs struct {
					AkfireInterconnectQuic *CDN `json:"akfire_interconnect_quic"`
					FastlySkyfire          *CDN `json:"fastly_skyfire"`
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
	// in case the video is not available anymore we still map the error message
	Message string `json:"message"`
}

type CDN struct {
	Url    string `json:"url"`
	Origin string `json:"origin"`
	AvcUrl string `json:"avc_url"`
}

func (m *vimeo) getPlayerJSON(item *models.TrackedItem) (*PlayerJson, error) {
	results := m.defaultVideoURLPattern.FindStringSubmatch(item.URI)
	if len(results) < 2 || len(results) > 4 {
		return nil, fmt.Errorf("unsupported URL format")
	}

	// handle https://player.vimeo.com/video/123456/abc123 URLs
	playerUrl := fmt.Sprintf("https://player.vimeo.com/video/%s/config", results[1])
	if len(results) == 3 && results[2] != "" {
		playerUrl += "?h=" + results[2]
	}

	// handle https://player.vimeo.com/video/123456?h=abc123 URLs
	if len(results) == 4 && results[3] != "" {
		playerUrl += "?h=" + results[3]
	}

	res, err := m.Session.Get(playerUrl)
	if err != nil {
		return nil, err
	}

	out, readErr := io.ReadAll(res.Body)
	if readErr != nil {
		return nil, readErr
	}

	var playerJson PlayerJson
	err = json.Unmarshal(out, &playerJson)

	return &playerJson, err
}

func (p *PlayerJson) GetMasterJSONUrl() string {
	// HLS is for apple products while MPEG-DASH is the international standard, so use dash by default
	if p.Request.Files.Dash.CDNs.AkfireInterconnectQuic != nil {
		return p.Request.Files.Dash.CDNs.AkfireInterconnectQuic.Url
	}

	if p.Request.Files.Dash.CDNs.FastlySkyfire != nil {
		return p.Request.Files.Dash.CDNs.FastlySkyfire.Url
	}

	// fallback to HLS CDN Urls
	if p.Request.Files.HLS.CDNs.AkfireInterconnectQuic != nil {
		return p.Request.Files.HLS.CDNs.AkfireInterconnectQuic.Url
	}

	if p.Request.Files.Dash.CDNs.FastlySkyfire != nil {
		return p.Request.Files.Dash.CDNs.FastlySkyfire.Url
	}

	return ""
}

func (p *PlayerJson) GetVideoTitle() string {
	return p.Video.Title
}
