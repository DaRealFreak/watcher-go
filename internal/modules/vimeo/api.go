package vimeo

import (
	"encoding/json"
	"io"
	"net/http"
)

type JWTResponse struct {
	Token string `json:"token"`
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
