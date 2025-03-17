package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

// Config holds the configuration for the OIDC flow.
type Config struct {
	InitialAuthURL string       // URL to initiate the OIDC flow.
	TokenURL       string       // Endpoint to log in and retrieve initial tokens.
	FinalizeURL    string       // Endpoint to exchange the auth code for the final token.
	ClientID       string       // Your client ID.
	RedirectURI    string       // Your redirect URI.
	HTTPClient     *http.Client // The HTTP client to use (with proxy, cookies, etc.).
}

// OidcClient encapsulates the OIDC flow implementation.
// This type is unexported so that only its methods (that we choose to export) are visible.
type OidcClient struct {
	cfg *Config
}

// authResponse represents the JSON response from /auth/token.
type authResponse struct {
	Success      bool   `json:"success"`
	TokenType    string `json:"token_type"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// finalizeResponse represents the JSON response from /sso/finalize.
type finalizeResponse struct {
	Success         bool   `json:"success"`
	TokenType       string `json:"token_type"`
	AccessToken     string `json:"access_token"`
	AccessTokenTTL  int    `json:"access_token_ttl"`
	RefreshToken    string `json:"refresh_token"`
	RefreshTokenTTL int    `json:"refresh_token_ttl"`
	PasswordHash    string `json:"password_hash"`
}

// NewOIDCClient creates and returns a new OIDC client using the provided configuration.
func NewOIDCClient(cfg Config) (*OidcClient, error) {
	if cfg.HTTPClient == nil {
		jar, err := cookiejar.New(nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create cookie jar: %w", err)
		}
		cfg.HTTPClient = &http.Client{Jar: jar}
	}
	return &OidcClient{cfg: &cfg}, nil
}

// GetOAuthToken performs the entire OIDC flow using the supplied username and password,
// and returns an oauth2.Token that can be used to access the API.
func (o *OidcClient) GetOAuthToken(ctx context.Context, username, password string) (*oauth2.Token, error) {
	interactionURL, err := o.startFlow(ctx)
	if err != nil {
		return nil, fmt.Errorf("error starting OIDC flow: %w", err)
	}

	ar, err := o.login(ctx, username, password)
	if err != nil {
		return nil, fmt.Errorf("error during login: %w", err)
	}

	code, err := o.finalizeInteraction(ctx, interactionURL, ar.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("error finalizing interaction: %w", err)
	}

	fr, err := o.retrieveFinalToken(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("error retrieving final token: %w", err)
	}

	token := &oauth2.Token{
		AccessToken:  fr.AccessToken,
		TokenType:    fr.TokenType,
		RefreshToken: fr.RefreshToken,
		Expiry:       time.Now().Add(time.Duration(fr.AccessTokenTTL) * time.Second),
	}
	return token, nil
}

// startFlow performs the initial GET request and extracts the interaction URL.
func (o *OidcClient) startFlow(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", o.cfg.InitialAuthURL, nil)
	if err != nil {
		return "", fmt.Errorf("error creating initial GET request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:136.0) Gecko/20100101 Firefox/136.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Sec-GPC", "1")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Sec-Fetch-Dest", "iframe")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "same-site")
	req.Header.Set("Priority", "u=4")
	req.Header.Set("Referer", "https://www.sankakucomplex.com/")

	resp, err := o.cfg.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error during initial GET request: %w", err)
	}
	defer resp.Body.Close()

	finalURL := resp.Request.URL.String()
	parsedURL, err := url.Parse(finalURL)
	if err != nil {
		return "", fmt.Errorf("error parsing final URL: %w", err)
	}
	submitURL := parsedURL.Query().Get("submit_url")
	if submitURL == "" {
		return "", fmt.Errorf("submit_url not found in final URL: %s", finalURL)
	}
	return "https://login.sankakucomplex.com" + submitURL, nil
}

// login sends the login POST request to /auth/token and returns an authResponse.
func (o *OidcClient) login(ctx context.Context, username, password string) (*authResponse, error) {
	payload := fmt.Sprintf(`{"login": "%s", "password": "%s", "mfaParams": {"login": "%s"}}`, username, password, username)
	req, err := http.NewRequestWithContext(ctx, "POST", o.cfg.TokenURL, bytes.NewBuffer([]byte(payload)))
	if err != nil {
		return nil, fmt.Errorf("error creating login POST request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:136.0) Gecko/20100101 Firefox/136.0")
	req.Header.Set("Accept", "application/vnd.sankaku.api+json;v=2")
	req.Header.Set("Accept-Language", "en-US")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Platform", "sso")
	req.Header.Set("Sec-GPC", "1")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Priority", "u=0")
	req.Header.Set("Referer", o.cfg.InitialAuthURL)

	resp, err := o.cfg.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error during login POST: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading login response: %w", err)
	}
	var ar authResponse
	if err = json.Unmarshal(body, &ar); err != nil {
		return nil, fmt.Errorf("error unmarshaling login response: %w", err)
	}
	if !ar.Success {
		return nil, fmt.Errorf("login failed, response: %s", string(body))
	}
	return &ar, nil
}

// finalizeInteraction posts the access token to the interaction URL and extracts the authorization code.
func (o *OidcClient) finalizeInteraction(ctx context.Context, interactionURL, accessToken string) (string, error) {
	formData := url.Values{}
	formData.Set("access_token", accessToken)
	formData.Set("state", "lang%3Den%26theme%3Dblack%26return_uri%3Dhttps%3A%2F%2Fwww.sankakucomplex.com%2F")

	req, err := http.NewRequestWithContext(ctx, "POST", interactionURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return "", fmt.Errorf("error creating interaction POST request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:136.0) Gecko/20100101 Firefox/136.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Sec-GPC", "1")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Sec-Fetch-Dest", "iframe")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Priority", "u=4")
	req.Header.Set("Referer", o.cfg.InitialAuthURL)

	resp, err := o.cfg.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error during interaction POST: %w", err)
	}
	defer resp.Body.Close()

	finalCallbackURL := resp.Request.URL.String()
	parsedCallback, err := url.Parse(finalCallbackURL)
	if err != nil {
		return "", fmt.Errorf("error parsing callback URL: %w", err)
	}
	code := parsedCallback.Query().Get("code")
	if code == "" {
		return "", fmt.Errorf("authorization code not found in callback URL: %s", finalCallbackURL)
	}
	return code, nil
}

// retrieveFinalToken exchanges the authorization code for the final token.
func (o *OidcClient) retrieveFinalToken(ctx context.Context, code string) (*finalizeResponse, error) {
	finalPayload := map[string]string{
		"code":         code,
		"client_id":    o.cfg.ClientID,
		"redirect_uri": o.cfg.RedirectURI,
	}
	jsonPayload, err := json.Marshal(finalPayload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling final payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", o.cfg.FinalizeURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("error creating final token request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:136.0) Gecko/20100101 Firefox/136.0")
	req.Header.Set("Accept", "application/vnd.sankaku.api+json;v=2")
	req.Header.Set("Accept-Language", "en-US,en")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Client-Type", "non-premium")
	req.Header.Set("Platform", "web-app")
	req.Header.Set("Api-Version", "2")
	req.Header.Set("Obfuscate-Type", "tag,wiki")
	req.Header.Set("Enable-New-Tag-Type", "true")
	req.Header.Set("Sec-GPC", "1")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	req.Header.Set("Referer", "https://www.sankakucomplex.com/")

	// The final token request omits cookies.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error during final token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading final token response: %w", err)
	}
	var fr finalizeResponse
	if err = json.Unmarshal(body, &fr); err != nil {
		return nil, fmt.Errorf("error unmarshaling final token response: %w", err)
	}
	if !fr.Success {
		return nil, fmt.Errorf("final token exchange failed, response: %s", string(body))
	}
	return &fr, nil
}
