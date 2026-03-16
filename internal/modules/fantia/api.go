package fantia

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	http "github.com/bogdanfinn/fhttp"

	"github.com/DaRealFreak/watcher-go/internal/raven"
)

var postIDPattern = regexp.MustCompile(`/posts/(\d+)`)

// postAPIResponse is the wrapper for the post API response
type postAPIResponse struct {
	Post postData `json:"post"`
}

type postData struct {
	ID           int           `json:"id"`
	Title        string        `json:"title"`
	Comment      string        `json:"comment"`
	PostContents []postContent `json:"post_contents"`
	Fanclub      fanclub       `json:"fanclub"`
	Thumb        *postThumb    `json:"thumb"`
}

type postThumb struct {
	Original string `json:"original"`
}

type postContent struct {
	ID                int                `json:"id"`
	Title             string             `json:"title"`
	Category          string             `json:"category"`
	VisibleStatus     string             `json:"visible_status"`
	Comment           string             `json:"comment"`
	Filename          string             `json:"filename"`
	DownloadURI       string             `json:"download_uri"`
	PostContentPhotos []postContentPhoto `json:"post_content_photos"`
	Plan              *contentPlan       `json:"plan"`
}

type contentPlan struct {
	ID    int    `json:"id"`
	Price int    `json:"price"`
	Name  string `json:"name"`
}

type postContentPhoto struct {
	ID  json.Number `json:"id"`
	URL photoURLs   `json:"url"`
}

type photoURLs struct {
	Original string `json:"original"`
	Main     string `json:"main"`
	Large    string `json:"large"`
	Medium   string `json:"medium"`
}

type fanclub struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	User struct {
		Name string `json:"name"`
	} `json:"user"`
}

// blog content types for parsing the comment JSON
type blogContent struct {
	Ops []blogOp `json:"ops"`
}

type blogOp struct {
	Insert json.RawMessage `json:"insert"`
}

type blogInsertImage struct {
	FantiaImage struct {
		ID          string `json:"id"`
		OriginalURL string `json:"original_url"`
	} `json:"fantiaImage"`
}

// newAPIRequest creates a request with the required headers for the fantia JSON API
func (m *fantia) newAPIRequest(apiURL string) (*http.Request, error) {
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header = http.Header{
		"accept":           {"application/json, text/plain, */*"},
		"x-requested-with": {"XMLHttpRequest"},
		"referer":          {"https://fantia.jp"},
		http.HeaderOrderKey: {
			"accept",
			"x-csrf-token",
			"x-requested-with",
			"referer",
		},
		http.PHeaderOrderKey: {
			":method",
			":authority",
			":scheme",
			":path",
		},
	}

	if m.csrfToken != "" {
		req.Header.Set("X-CSRF-Token", m.csrfToken)
	}

	return req, nil
}

// ensureCSRFToken fetches the CSRF token from the fantia homepage if not already set
func (m *fantia) ensureCSRFToken() error {
	if m.csrfToken != "" {
		return nil
	}

	resp, err := m.Session.Get("https://fantia.jp")
	if err != nil {
		return err
	}

	doc := m.Session.GetDocument(resp)
	if token, exists := doc.Find("meta[name=csrf-token]").Attr("content"); exists {
		m.csrfToken = token
	}

	return nil
}

// getPost fetches an individual post with full details including content and photos
func (m *fantia) getPost(postID string) (*postData, error) {
	if err := m.ensureCSRFToken(); err != nil {
		return nil, err
	}

	apiURL := fmt.Sprintf("https://fantia.jp/api/v1/posts/%s", postID)

	req, err := m.newAPIRequest(apiURL)
	if err != nil {
		return nil, err
	}

	resp, err := m.Session.GetClient().Do(req)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	raven.CheckClosure(resp.Body)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get post %s: status %d", postID, resp.StatusCode)
	}

	var apiResp postAPIResponse
	if err = json.Unmarshal(body, &apiResp); err != nil {
		return nil, err
	}

	return &apiResp.Post, nil
}

// getPostIDs fetches an HTML page of the fanclub's posts and extracts post IDs
func (m *fantia) getPostIDs(fanclubID string, page int) ([]string, error) {
	pageURL := fmt.Sprintf("https://fantia.jp/fanclubs/%s/posts?page=%d&q[s]=newer", fanclubID, page)

	resp, err := m.Session.Get(pageURL)
	if err != nil {
		return nil, err
	}

	doc := m.Session.GetDocument(resp)

	// extract CSRF token from the page while we're at it
	if token, exists := doc.Find("meta[name=csrf-token]").Attr("content"); exists {
		m.csrfToken = token
	}

	// extract unique post IDs from all links
	seen := make(map[string]bool)
	var ids []string

	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}
		matches := postIDPattern.FindStringSubmatch(href)
		if len(matches) > 1 {
			id := matches[1]
			if !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
	})

	return ids, nil
}

// getBestPhotoURL returns the highest quality URL available for a photo
func getBestPhotoURL(photo postContentPhoto) string {
	if photo.URL.Original != "" {
		return photo.URL.Original
	}
	if photo.URL.Main != "" {
		return photo.URL.Main
	}
	if photo.URL.Large != "" {
		return photo.URL.Large
	}
	return photo.URL.Medium
}

// extractBlogImages parses blog JSON content and extracts embedded image URLs
func extractBlogImages(comment string) []string {
	var blog blogContent
	if err := json.Unmarshal([]byte(comment), &blog); err != nil {
		return nil
	}

	var urls []string
	for _, op := range blog.Ops {
		var img blogInsertImage
		if err := json.Unmarshal(op.Insert, &img); err == nil && img.FantiaImage.OriginalURL != "" {
			imgURL := img.FantiaImage.OriginalURL
			if strings.HasPrefix(imgURL, "/") {
				imgURL = "https://fantia.jp" + imgURL
			}
			urls = append(urls, imgURL)
		}
	}

	return urls
}
