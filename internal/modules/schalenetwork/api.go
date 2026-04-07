package schalenetwork

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"strings"

	http "github.com/bogdanfinn/fhttp"
	"github.com/DaRealFreak/watcher-go/internal/http/tls_session"
)

type bookDetailResponse struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Tags      []bookTag `json:"tags"`
	CreatedAt int64     `json:"created_at"`
}

type bookTag struct {
	Name      string `json:"name"`
	Namespace int    `json:"namespace"`
}

type bookDataResponse struct {
	Data   map[string]bookFormat `json:"data"`
	Source string                `json:"source"`
}

type bookFormat struct {
	ID  json.Number `json:"id"`
	Key string      `json:"key"`
}

type bookImageListResponse struct {
	Base    string           `json:"base"`
	Entries []bookImageEntry `json:"entries"`
}

type bookImageEntry struct {
	Path       string `json:"path"`
	Dimensions []int  `json:"dimensions"`
}

type searchResponse struct {
	Entries []searchEntry `json:"entries"`
	Limit   int           `json:"limit"`
	Page    int           `json:"page"`
	Total   int           `json:"total"`
}

type searchEntry struct {
	ID    int    `json:"id"`
	Key   string `json:"key"`
	Title string `json:"title"`
}

const (
	tagNamespaceArtist   = 1
	tagNamespaceLanguage = 11
)

func (r *bookDetailResponse) GetTitle() string {
	return r.Title
}

func (r *bookDetailResponse) GetLanguage() string {
	for _, tag := range r.Tags {
		if tag.Namespace == tagNamespaceLanguage {
			if tag.Name == "translated" {
				continue
			}

			if len(tag.Name) > 0 {
				return strings.ToUpper(tag.Name[:1]) + tag.Name[1:]
			}
		}
	}

	return ""
}

func (r *bookDetailResponse) GetArtist() string {
	for _, tag := range r.Tags {
		if tag.Namespace == tagNamespaceArtist {
			return tag.Name
		}
	}

	return ""
}

func (m *schaleNetwork) getBookDetail(id, key string) (*bookDetailResponse, error) {
	apiURL := fmt.Sprintf("%s/books/detail/%s/%s", m.apiBaseURL(), id, key)

	res, err := m.get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get book detail: %w", err)
	}

	var apiRes bookDetailResponse
	if err = m.mapAPIResponse(res, &apiRes); err != nil {
		return nil, fmt.Errorf("failed to map book detail response: %w", err)
	}

	return &apiRes, nil
}

// tryClearanceWithSession attempts clearance validation using the given session
func (m *schaleNetwork) tryClearanceWithSession(session *tls_session.TlsClientSession) error {
	authURL := fmt.Sprintf("%s/clearance", m.authBaseURL())

	req, err := http.NewRequest("GET", authURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create clearance request: %w", err)
	}

	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", m.siteBaseURL()+"/")
	req.Header.Set("Origin", m.siteBaseURL())
	req.Header.Set("Authorization", "Bearer "+m.crt)
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "cross-site")

	if m.settings.Cloudflare.UserAgent != "" {
		req.Header.Set("User-Agent", m.settings.Cloudflare.UserAgent)
	}

	// call the underlying client directly to bypass error handlers and avoid recursion
	res, err := session.Client.Do(req)
	if err != nil {
		return err
	}

	if res.Body != nil {
		res.Body.Close()
	}

	if res.StatusCode == 403 {
		return tls_session.StatusError{StatusCode: 403}
	}

	return nil
}

// rotateToClearProxy tries clearance with each available proxy until one succeeds.
// On success it switches the main session to the working proxy.
func (m *schaleNetwork) rotateToClearProxy() error {
	mainSession, ok := m.Session.(*tls_session.TlsClientSession)
	if !ok {
		return fmt.Errorf("main session is not a TlsClientSession")
	}

	// reset proxy exclusions so previously blocked proxies are retried
	for _, proxy := range m.proxies {
		proxy.excluded = false
	}

	// try clearance with the current main session first
	err := m.tryClearanceWithSession(mainSession)
	if err == nil {
		slog.Debug("clearance validated with current proxy", "module", m.Key)
		m.clearanceValidated = true
		return nil
	}

	if !m.isStatusError(err, http.StatusForbidden) {
		// non-403 error, return it directly
		return fmt.Errorf("clearance validation failed: %w", err)
	}

	slog.Warn("current proxy returned 403 on clearance", "module", m.Key)

	// no multi-proxy available, cannot rotate
	if !m.settings.MultiProxy || len(m.proxies) == 0 {
		return fmt.Errorf("proxy/VPN appears blocked (clearance returned 403)")
	}

	// try each non-excluded proxy session
	for _, proxy := range m.proxies {
		if proxy.excluded {
			continue
		}

		slog.Info(fmt.Sprintf("trying clearance with proxy %s", proxy.proxy.Host), "module", m.Key)
		err = m.tryClearanceWithSession(proxy.session)
		if err == nil {
			slog.Info(fmt.Sprintf("clearance succeeded with proxy %s, switching main session", proxy.proxy.Host), "module", m.Key)
			if setErr := m.Session.SetProxy(&proxy.proxy); setErr != nil {
				return fmt.Errorf("failed to switch proxy: %w", setErr)
			}

			m.clearanceValidated = true
			return nil
		}

		if m.isStatusError(err, http.StatusForbidden) {
			slog.Warn(fmt.Sprintf("proxy %s returned 403 on clearance, excluding", proxy.proxy.Host), "module", m.Key)
			proxy.excluded = true
			continue
		}

		// non-403 error on this proxy, skip it
		slog.Warn(fmt.Sprintf("proxy %s clearance failed: %s", proxy.proxy.Host, err.Error()), "module", m.Key)
	}

	return fmt.Errorf("all proxies excluded due to 403 errors")
}

// recoverFrom403 handles 403 recovery.
// If clearance was already validated (crt previously accepted), try proxy rotation first
// since the most common cause is the current VPN/proxy getting blacklisted.
// If clearance was never validated (startup), prompt for a new crt token first.
func (m *schaleNetwork) recoverFrom403() error {
	if m.clearanceValidated {
		slog.Warn("received 403 after successful clearance, trying proxy rotation first...", "module", m.Key)
		if err := m.rotateToClearProxy(); err == nil {
			return nil
		}

		slog.Warn("proxy rotation failed, requesting new crt token...", "module", m.Key)
	} else {
		slog.Warn("received 403, requesting new crt token...", "module", m.Key)
	}

	m.promptCrtRefresh()

	if m.crt == "" {
		return fmt.Errorf("no crt token provided")
	}

	return m.rotateToClearProxy()
}

func (m *schaleNetwork) getBookData(id, key string) (*bookDataResponse, error) {
	if m.crt == "" {
		slog.Warn("crt token is not set, requesting token now...", "module", m.Key)
		m.promptCrtRefresh()

		if m.crt == "" {
			return nil, fmt.Errorf(
				"crt token is required for downloading images, " +
					"set it in config (Modules.niyaniya_moe.crt) or as a cookie",
			)
		}
	}

	apiURL := fmt.Sprintf("%s/books/detail/%s/%s?crt=%s", m.apiBaseURL(), id, key, m.crt)

	res, err := m.post(apiURL)
	if m.isStatusError(err, http.StatusForbidden) {
		if recoverErr := m.recoverFrom403(); recoverErr != nil {
			return nil, fmt.Errorf("failed to get book data: %w", recoverErr)
		}

		apiURL = fmt.Sprintf("%s/books/detail/%s/%s?crt=%s", m.apiBaseURL(), id, key, m.crt)
		res, err = m.post(apiURL)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get book data: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf(
			"failed to get book data: unexpected status code: %d. response: %s",
			res.StatusCode, string(body),
		)
	}

	var apiRes bookDataResponse
	if err = m.mapAPIResponse(res, &apiRes); err != nil {
		return nil, fmt.Errorf("failed to map book data response: %w", err)
	}

	return &apiRes, nil
}

func (m *schaleNetwork) getBookImages(bookID, bookKey, fmtID, fmtKey, fmtW string) (*bookImageListResponse, error) {
	apiURL := fmt.Sprintf(
		"%s/books/data/%s/%s/%s/%s/%s?crt=%s",
		m.apiBaseURL(), bookID, bookKey, fmtID, fmtKey, fmtW, m.crt,
	)

	res, err := m.get(apiURL)
	if m.isStatusError(err, http.StatusForbidden) {
		if recoverErr := m.recoverFrom403(); recoverErr != nil {
			return nil, fmt.Errorf("failed to get book images: %w", recoverErr)
		}

		apiURL = fmt.Sprintf(
			"%s/books/data/%s/%s/%s/%s/%s?crt=%s",
			m.apiBaseURL(), bookID, bookKey, fmtID, fmtKey, fmtW, m.crt,
		)
		res, err = m.get(apiURL)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get book images: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get book images: status code %d", res.StatusCode)
	}

	var apiRes bookImageListResponse
	if err = m.mapAPIResponse(res, &apiRes); err != nil {
		return nil, fmt.Errorf("failed to map book images response: %w", err)
	}

	return &apiRes, nil
}

func (m *schaleNetwork) getSearch(query string, page int, extraParams url.Values) (*searchResponse, error) {
	encoded := url.QueryEscape(query)
	// restore search syntax characters that the API expects unencoded
	encoded = strings.NewReplacer("%3A", ":", "%5E", "^", "%24", "$").Replace(encoded)
	apiURL := fmt.Sprintf(
		"%s/books?s=%s&page=%d",
		m.apiBaseURL(), encoded, page,
	)

	// append extra query parameters (e.g. lang)
	for key, values := range extraParams {
		for _, value := range values {
			apiURL += fmt.Sprintf("&%s=%s", url.QueryEscape(key), url.QueryEscape(value))
		}
	}

	res, err := m.get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get search results: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get search results: status code %d", res.StatusCode)
	}

	var apiRes searchResponse
	if err = m.mapAPIResponse(res, &apiRes); err != nil {
		return nil, fmt.Errorf("failed to map search response: %w", err)
	}

	return &apiRes, nil
}

// isStatusError checks if the error is a StatusError with the given status code
func (m *schaleNetwork) isStatusError(err error, statusCode int) bool {
	var statusErr tls_session.StatusError
	return errors.As(err, &statusErr) && statusErr.StatusCode == statusCode
}

// mapAPIResponse maps the API response into the passed APIResponse type
func (m *schaleNetwork) mapAPIResponse(res *http.Response, apiRes interface{}) error {
	out, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	content := string(out)

	if res.StatusCode >= 400 {
		return fmt.Errorf(`unknown error response: "%s"`, content)
	}

	if err = json.Unmarshal([]byte(content), &apiRes); err != nil {
		return err
	}

	return nil
}
