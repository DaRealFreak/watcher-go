package api

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/DaRealFreak/watcher-go/internal/http"
)

// CustomTime for parsing timestamps with fractional seconds
type CustomTime struct {
	time.Time
}

func (ct *CustomTime) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		return nil
	}

	s := string(b[1 : len(b)-1]) // Remove quotes
	layouts := []string{
		"2006-01-02T15:04:05.000000",
		"2006-01-02T15:04:05",
	}
	var err error
	for _, layout := range layouts {
		ct.Time, err = time.Parse(layout, s)
		if err == nil {
			return nil
		}
	}
	return fmt.Errorf("could not parse time: %s, error: %w", s, err)
}

// Client manages communication with the Kemono API
type Client struct {
	BaseURL string
	Client  http.TlsClientSessionInterface
}

// NewClient returns a new Kemono API client
func NewClient(baseURL string, client http.TlsClientSessionInterface) *Client {
	return &Client{
		BaseURL: baseURL,
		Client:  client,
	}
}

func (api *Client) GetUserProfile(service string, userID string) (*Profile, error) {
	apiURL := fmt.Sprintf("%s/api/v1/%s/user/%s/profile", api.BaseURL, service, userID)
	resp, err := api.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var root Profile
	err = json.Unmarshal(body, &root)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	return &root, nil
}

// GetUserPosts fetches user posts from the API
func (api *Client) GetUserPosts(service, userID string, offset int) ([]QuickPost, error) {
	apiURL := fmt.Sprintf("%s/api/v1/%s/user/%s/posts", api.BaseURL, service, userID)
	if offset > 0 {
		apiURL = fmt.Sprintf("%s?o=%d", apiURL, offset)
	}
	resp, err := api.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var posts []QuickPost
	if err = json.NewDecoder(resp.Body).Decode(&posts); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON into []QuickPost: %w", err)
	}

	return posts, nil
}

// GetPostDetails fetches post details from the API
func (api *Client) GetPostDetails(service, userID, postID string) (*PostRoot, error) {
	apiURL := fmt.Sprintf("%s/api/v1/%s/user/%s/post/%s", api.BaseURL, service, userID, postID)
	resp, err := api.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var postRoot PostRoot
	err = json.Unmarshal(body, &postRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	return &postRoot, nil
}

func (api *Client) GetPostComments(service, userID, postID string) (comments []Comment, err error) {
	apiURL := fmt.Sprintf("%s/api/v1/%s/user/%s/post/%s/comments", api.BaseURL, service, userID, postID)
	resp, err := api.Get(apiURL)
	// normal behavior if no comments are available is a 404 response
	if resp != nil && resp.StatusCode == 404 {
		return comments, nil
	}

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &comments)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	return comments, nil
}
