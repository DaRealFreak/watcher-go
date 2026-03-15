package bsky

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	http "github.com/bogdanfinn/fhttp"

	"github.com/DaRealFreak/watcher-go/internal/raven"
)

const apiBaseURL = "https://public.api.bsky.app/xrpc"

type profileResponse struct {
	DID         string `json:"did"`
	Handle      string `json:"handle"`
	DisplayName string `json:"displayName"`
}

type authorFeedResponse struct {
	Cursor string     `json:"cursor"`
	Feed   []feedItem `json:"feed"`
}

type feedItem struct {
	Post   postView        `json:"post"`
	Reason json.RawMessage `json:"reason"`
}

type postView struct {
	URI    string          `json:"uri"`
	CID    string          `json:"cid"`
	Author authorView      `json:"author"`
	Embed  json.RawMessage `json:"embed"`
}

type authorView struct {
	DID         string `json:"did"`
	Handle      string `json:"handle"`
	DisplayName string `json:"displayName"`
}

type embedType struct {
	Type string `json:"$type"`
}

type imagesEmbedView struct {
	Type   string      `json:"$type"`
	Images []imageView `json:"images"`
}

type imageView struct {
	Thumb    string `json:"thumb"`
	Fullsize string `json:"fullsize"`
	Alt      string `json:"alt"`
}

type videoEmbedView struct {
	Type      string `json:"$type"`
	CID       string `json:"cid"`
	Playlist  string `json:"playlist"`
	Thumbnail string `json:"thumbnail"`
}

type recordWithMediaEmbedView struct {
	Type  string          `json:"$type"`
	Media json.RawMessage `json:"media"`
}

type didDocument struct {
	Service []didService `json:"service"`
}

type didService struct {
	ID              string `json:"id"`
	Type            string `json:"type"`
	ServiceEndpoint string `json:"serviceEndpoint"`
}

type createSessionResponse struct {
	DID        string `json:"did"`
	Handle     string `json:"handle"`
	AccessJwt  string `json:"accessJwt"`
	RefreshJwt string `json:"refreshJwt"`
}

// getAPIBaseURL returns the appropriate XRPC base URL depending on auth state
func (m *bsky) getAPIBaseURL() string {
	if m.LoggedIn && m.authPDS != "" {
		return m.authPDS + "/xrpc"
	}
	return apiBaseURL
}

// apiGet makes a GET request, adding the Authorization header if authenticated
func (m *bsky) apiGet(apiURL string) (*http.Response, error) {
	if m.accessToken != "" {
		req, err := http.NewRequest("GET", apiURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+m.accessToken)
		return m.Session.Do(req)
	}
	return m.Session.Get(apiURL)
}

func (m *bsky) getProfile(actor string) (*profileResponse, error) {
	apiURL := fmt.Sprintf("%s/app.bsky.actor.getProfile?actor=%s",
		m.getAPIBaseURL(), url.QueryEscape(actor))

	resp, err := m.apiGet(apiURL)
	if err != nil {
		return nil, err
	}
	defer raven.CheckClosure(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get profile for %s: %s", actor, string(body))
	}

	var profile profileResponse
	if err = json.Unmarshal(body, &profile); err != nil {
		return nil, err
	}

	return &profile, nil
}

func (m *bsky) getAuthorFeed(actor string, cursor string) (*authorFeedResponse, error) {
	apiURL := fmt.Sprintf("%s/app.bsky.feed.getAuthorFeed?actor=%s&limit=100&filter=posts_with_media",
		m.getAPIBaseURL(), url.QueryEscape(actor))

	if cursor != "" {
		apiURL += "&cursor=" + url.QueryEscape(cursor)
	}

	resp, err := m.apiGet(apiURL)
	if err != nil {
		return nil, err
	}
	defer raven.CheckClosure(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get author feed for %s: %s", actor, string(body))
	}

	var feed authorFeedResponse
	if err = json.Unmarshal(body, &feed); err != nil {
		return nil, err
	}

	return &feed, nil
}

func (m *bsky) resolvePDS(did string) (string, error) {
	if !strings.HasPrefix(did, "did:plc:") {
		return m.settings.PDS, nil
	}

	resp, err := m.Session.Get(fmt.Sprintf("https://plc.directory/%s", did))
	if err != nil {
		return "", err
	}
	defer raven.CheckClosure(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to resolve DID %s: %s", did, string(body))
	}

	var doc didDocument
	if err = json.Unmarshal(body, &doc); err != nil {
		return "", err
	}

	for _, svc := range doc.Service {
		if svc.ID == "#atproto_pds" {
			return svc.ServiceEndpoint, nil
		}
	}

	return "", fmt.Errorf("no PDS found for DID %s", did)
}

// createAuthSession performs a fresh login using com.atproto.server.createSession
func (m *bsky) createAuthSession(identifier, password string) error {
	pdsURL := m.settings.PDS

	// resolve the correct PDS for handles and DIDs (emails go straight to the configured PDS)
	if strings.HasPrefix(identifier, "did:") {
		if resolved, resolveErr := m.resolvePDS(identifier); resolveErr == nil {
			pdsURL = resolved
		}
	} else if !strings.Contains(identifier, "@") {
		// handle (not email) — resolve via getProfile → DID → PDS
		if profile, profileErr := m.getProfile(identifier); profileErr == nil {
			if resolved, resolveErr := m.resolvePDS(profile.DID); resolveErr == nil {
				pdsURL = resolved
			}
		}
	}

	reqBody, err := json.Marshal(map[string]string{
		"identifier": identifier,
		"password":   password,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", pdsURL+"/xrpc/com.atproto.server.createSession", bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.Session.Do(req)
	if err != nil {
		return err
	}
	defer raven.CheckClosure(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login failed: %s", string(body))
	}

	var session createSessionResponse
	if err = json.Unmarshal(body, &session); err != nil {
		return err
	}

	m.accessToken = session.AccessJwt
	m.refreshToken = session.RefreshJwt
	m.authPDS = pdsURL

	m.storeCookie("access_jwt", session.AccessJwt, parseJWTExpiry(session.AccessJwt))
	m.storeCookie("refresh_jwt", session.RefreshJwt, parseJWTExpiry(session.RefreshJwt))
	m.storeCookie("auth_pds", pdsURL, "")

	return nil
}

// doRefreshSession refreshes the access token using com.atproto.server.refreshSession
func (m *bsky) doRefreshSession() error {
	req, err := http.NewRequest("POST", m.authPDS+"/xrpc/com.atproto.server.refreshSession", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+m.refreshToken)

	resp, err := m.Session.Do(req)
	if err != nil {
		return err
	}
	defer raven.CheckClosure(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token refresh failed: %s", string(body))
	}

	var session createSessionResponse
	if err = json.Unmarshal(body, &session); err != nil {
		return err
	}

	m.accessToken = session.AccessJwt
	m.refreshToken = session.RefreshJwt

	m.storeCookie("access_jwt", session.AccessJwt, parseJWTExpiry(session.AccessJwt))
	m.storeCookie("refresh_jwt", session.RefreshJwt, parseJWTExpiry(session.RefreshJwt))

	return nil
}

// storeCookie creates or updates a cookie in the database
func (m *bsky) storeCookie(name, value, expiration string) {
	if existing := m.DbIO.GetCookie(name, m); existing != nil {
		m.DbIO.UpdateCookie(name, value, expiration, m)
	} else {
		m.DbIO.GetFirstOrCreateCookie(name, value, expiration, m)
	}
}

// parseJWTExpiry extracts the expiration time from a JWT token and returns it as RFC3339 string
func parseJWTExpiry(token string) string {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return ""
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}

	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err = json.Unmarshal(payload, &claims); err != nil {
		return ""
	}

	return time.Unix(claims.Exp, 0).Format(time.RFC3339)
}
