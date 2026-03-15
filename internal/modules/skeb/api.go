package skeb

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"strings"

	http "github.com/bogdanfinn/fhttp"

	"github.com/DaRealFreak/watcher-go/internal/raven"
)

const skebAPIBase = "https://skeb.jp/api"

// workListItem is the minimal response from the works list endpoint (no previews)
type workListItem struct {
	Path    string `json:"path"`
	Private bool   `json:"private"`
}

// workResponse is the full response from the individual work endpoint (includes previews)
type workResponse struct {
	ID              int       `json:"id"`
	Path            string    `json:"path"`
	Private         bool      `json:"private"`
	NSFW            bool      `json:"nsfw"`
	Body            string    `json:"body"`
	Genre           string    `json:"genre"`
	Creator         skebUser  `json:"creator"`
	Previews        []preview `json:"previews"`
	OGImageURL      string    `json:"og_image_url"`
	ArticleImageURL string    `json:"article_image_url"`
}

type skebUser struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	ScreenName string `json:"screen_name"`
	AvatarURL  string `json:"avatar_url"`
}

type preview struct {
	ID          int         `json:"id"`
	URL         string      `json:"url"`
	Information previewInfo `json:"information"`
}

type previewInfo struct {
	Width     int     `json:"width"`
	Height    int     `json:"height"`
	ByteSize  int     `json:"byte_size"`
	Duration  float64 `json:"duration"`
	Extension string  `json:"extension"`
	IsMovie   bool    `json:"is_movie"`
}

func (m *skeb) newRequest(apiURL string) (*http.Request, error) {
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header = http.Header{
		"accept":          {"application/json, text/plain, */*"},
		"accept-language": {"en-US,en;q=0.9"},
		"authorization":   {"Bearer null"},
		"referer":         {"https://skeb.jp"},
		"sec-fetch-dest":  {"empty"},
		"sec-fetch-mode":  {"cors"},
		"sec-fetch-site":  {"same-origin"},
		"sec-gpc":         {"1"},
		"priority":        {"u=0"},
		http.HeaderOrderKey: {
			"accept",
			"accept-language",
			"authorization",
			"referer",
			"sec-fetch-dest",
			"sec-fetch-mode",
			"sec-fetch-site",
			"sec-gpc",
			"priority",
		},
		http.PHeaderOrderKey: {
			":method",
			":authority",
			":scheme",
			":path",
		},
	}

	return req, nil
}

func (m *skeb) apiGet(apiURL string) ([]byte, error) {
	for attempt := 0; attempt < 3; attempt++ {
		req, err := m.newRequest(apiURL)
		if err != nil {
			return nil, err
		}

		// bypass the session's error handler which treats 403 as fatal
		resp, err := m.Session.GetClient().Do(req)
		if err != nil {
			return nil, err
		}

		body, err := io.ReadAll(resp.Body)
		raven.CheckClosure(resp.Body)

		if err != nil {
			return nil, err
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			slog.Warn("received 429, extracting request_key and retrying", "module", m.Key)
			m.handle429(resp, body)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
		}

		return body, nil
	}

	return nil, fmt.Errorf("request failed after retries (429 rate limited)")
}

// getWorksList returns the list of works (without previews) for pagination
func (m *skeb) getWorksList(username string, offset int) ([]workListItem, error) {
	apiURL := fmt.Sprintf("%s/users/%s/works?role=%s&sort=date&offset=%d",
		skebAPIBase, url.PathEscape(username), url.QueryEscape(m.settings.Role), offset)

	body, err := m.apiGet(apiURL)
	if err != nil {
		return nil, err
	}

	var works []workListItem
	if err = json.Unmarshal(body, &works); err != nil {
		return nil, err
	}

	return works, nil
}

// getWork fetches an individual work with full details including previews
func (m *skeb) getWork(username string, postNum string) (*workResponse, error) {
	apiURL := fmt.Sprintf("%s/users/%s/works/%s",
		skebAPIBase, url.PathEscape(username), url.PathEscape(postNum))

	body, err := m.apiGet(apiURL)
	if err != nil {
		return nil, err
	}

	var work workResponse
	if err = json.Unmarshal(body, &work); err != nil {
		return nil, err
	}

	return &work, nil
}

func (m *skeb) handle429(resp *http.Response, body []byte) {
	skebURL, _ := url.Parse("https://skeb.jp")

	// check response cookies for request_key
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "request_key" {
			m.Session.GetClient().SetCookies(skebURL, []*http.Cookie{
				{Name: "request_key", Value: cookie.Value, Domain: "skeb.jp"},
			})
			return
		}
	}

	// extract from response body
	bodyStr := string(body)
	if idx := strings.Index(bodyStr, "request_key="); idx >= 0 {
		value := bodyStr[idx+len("request_key="):]
		if end := strings.Index(value, ";"); end >= 0 {
			value = value[:end]
		}
		m.Session.GetClient().SetCookies(skebURL, []*http.Cookie{
			{Name: "request_key", Value: value, Domain: "skeb.jp"},
		})
	}
}
