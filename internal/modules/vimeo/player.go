package vimeo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"io"
	"net/http"
	"regexp"
)

type MicroDataJson []struct {
	Name     string `json:"name"`
	EmbedUrl string `json:"embedUrl"`
}

type UrlInfo struct {
	VideoID string
	H       string
}

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

type PasswordRequest struct {
	Password string `json:"password"`
	Token    string `json:"token"`
}

func (m *vimeo) getPasswordProtectedPlayerJSON(
	item *models.TrackedItem,
	requiredPassword string,
) (*PlayerJson, error) {
	urlInfo, err := m.getUrlInfo(item.URI)
	if err != nil {
		return nil, err
	}

	xsrf, xsrfErr := m.getXSRFToken(item.URI)
	if xsrfErr != nil {
		return nil, xsrfErr
	}

	token, jwtErr := m.getJWT(item.URI)
	if jwtErr != nil {
		return nil, jwtErr
	}

	passwordUrl := fmt.Sprintf("https://vimeo.com/%s/password", urlInfo.VideoID)
	passwordRequest := &PasswordRequest{
		Password: requiredPassword,
		Token:    xsrf.XSRFToken,
	}

	body, jsonEncodeError := json.Marshal(passwordRequest)
	if jsonEncodeError != nil {
		return nil, jsonEncodeError
	}

	req, _ := http.NewRequest("POST", passwordUrl, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("jwt %s", token.Token))

	client := m.Session.GetClient()
	res, requestErr := client.Do(req)

	if requestErr != nil {
		return nil, requestErr
	}

	switch res.StatusCode {
	case http.StatusNotFound:
		return nil, fmt.Errorf("video not found")
	case http.StatusForbidden:
		return nil, fmt.Errorf("no access to video")
	}

	out, readErr := io.ReadAll(res.Body)
	if readErr != nil {
		return nil, readErr
	}

	var microDataJson MicroDataJson
	microdata := regexp.MustCompile(`(?m)<script id="microdata" type="application/ld\+json">([^<]+)</script>`).FindSubmatch(out)[1]
	err = json.Unmarshal(microdata, &microDataJson)
	if err != nil {
		return nil, err
	}

	if len(microDataJson) == 0 || microDataJson[0].EmbedUrl == "" {
		return nil, fmt.Errorf("unable to extract microdata from video page")
	}

	return m.getPlayerJSON(microDataJson[0].EmbedUrl)
}

func (m *vimeo) getPlayerJSON(itemUri string) (*PlayerJson, error) {
	urlInfo, urlErr := m.getUrlInfo(itemUri)
	if urlErr != nil {
		return nil, urlErr
	}

	token, err := m.getJWT(itemUri)
	if err != nil {
		return nil, err
	}

	var playerJson PlayerJson
	playerUrl := fmt.Sprintf("https://player.vimeo.com/video/%s", urlInfo.VideoID)
	if urlInfo.H != "" {
		playerUrl += fmt.Sprintf("?h=%s", urlInfo.H)
	}

	client := m.Session.GetClient()
	req, _ := http.NewRequest("GET", playerUrl, nil)

	req.Header.Set("Authorization", fmt.Sprintf("jwt %s", token.Token))

	res, requestErr := client.Do(req)

	if requestErr != nil {
		return nil, requestErr
	}

	switch res.StatusCode {
	case http.StatusNotFound:
		return nil, fmt.Errorf("video not found")
	case http.StatusForbidden:
		return nil, fmt.Errorf("no access to video")
	}

	out, readErr := io.ReadAll(res.Body)
	if readErr != nil {
		return nil, readErr
	}

	playerConfig := regexp.MustCompile(`<script>window.playerConfig =(.*?)</script>`).FindSubmatch(out)[1]

	err = json.Unmarshal(playerConfig, &playerJson)

	return &playerJson, err
}

func (m *vimeo) getUrlInfo(itemUri string) (*UrlInfo, error) {
	results := m.defaultVideoURLPattern.FindStringSubmatch(itemUri)
	if len(results) < 2 || len(results) > 4 {
		return nil, fmt.Errorf("unsupported URL format")
	}

	var videoID string
	var h string

	videoID = results[1]

	// handle https://player.vimeo.com/video/123456/abc123 URLs
	if results[2] != "" {
		h = results[2]
	}

	// handle https://player.vimeo.com/video/123456?h=abc123 URLs
	if len(results) == 4 && results[3] != "" {
		h = results[3]
	}

	return &UrlInfo{
		VideoID: videoID,
		H:       h,
	}, nil
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
