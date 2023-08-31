package vimeo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type JWTResponse struct {
	Token string `json:"token"`
}

type ApiVideoInfoResponse struct {
	EmbedPlayerConfigUrl string `json:"embed_player_config_url"`
}

// getApiURL returns the API URL for the video
func (m *vimeo) getApiURL(video string, handle string) string {
	// https://api.vimeo.com/videos/842345781:c201b314b9?fields=embed_player_config_url
	values := url.Values{
		"fields": {
			"embed_player_config_url",
		},
	}

	if handle != "" {
		return fmt.Sprintf("https://api.vimeo.com/videos/%s:%s?%s", video, handle, values.Encode())
	} else {
		return fmt.Sprintf("https://api.vimeo.com/videos/%s?%s", video, values.Encode())
	}
}

// getJWT returns the JWT token for the API requests
func (m *vimeo) getJWT(referer string) (*JWTResponse, error) {
	client := m.Session.GetClient()
	req, _ := http.NewRequest("GET", "https://vimeo.com/_next/jwt", nil)

	req.Header.Set("Referrer", referer)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	out, readErr := io.ReadAll(res.Body)
	if readErr != nil {
		return nil, readErr
	}

	var jwtResponse JWTResponse
	err = json.Unmarshal(out, &jwtResponse)

	return &jwtResponse, err
}

// getVideoApiInfo returns the API video info
func (m *vimeo) getVideoApiInfo(apiUrl string, jwtToken *JWTResponse) (*ApiVideoInfoResponse, error) {
	client := m.Session.GetClient()
	req, _ := http.NewRequest("GET", apiUrl, nil)

	req.Header.Set("Authorization", fmt.Sprintf("jwt %s", jwtToken.Token))

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	out, readErr := io.ReadAll(res.Body)
	if readErr != nil {
		return nil, readErr
	}

	var apiVideoInfoResponse ApiVideoInfoResponse
	err = json.Unmarshal(out, &apiVideoInfoResponse)

	return &apiVideoInfoResponse, err
}
