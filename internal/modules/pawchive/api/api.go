package api

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/DaRealFreak/watcher-go/internal/http"
)

// Client manages communication with the pawchive API.
type Client struct {
	BaseURL string
	Client  http.TlsClientSessionInterface
}

// NewClient returns a new pawchive API client.
func NewClient(baseURL string, client http.TlsClientSessionInterface) *Client {
	return &Client{BaseURL: baseURL, Client: client}
}

func (api *Client) GetUserProfile(service, userID string) (*Profile, error) {
	apiURL := fmt.Sprintf("%s/api/v1/%s/user/%s/profile", api.BaseURL, service, userID)
	resp, err := api.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var root Profile
	if err = json.Unmarshal(body, &root); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	return &root, nil
}

func (api *Client) GetUserPosts(service, userID string, offset int) ([]Post, error) {
	apiURL := fmt.Sprintf("%s/api/v1/%s/user/%s/posts", api.BaseURL, service, userID)
	if offset > 0 {
		apiURL = fmt.Sprintf("%s?o=%d", apiURL, offset)
	}
	resp, err := api.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var posts []Post
	if err = json.NewDecoder(resp.Body).Decode(&posts); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON into []Post: %w", err)
	}
	return posts, nil
}

func (api *Client) GetPostDetails(service, userID, postID string) (*Post, error) {
	apiURL := fmt.Sprintf("%s/api/v1/%s/user/%s/post/%s", api.BaseURL, service, userID, postID)
	resp, err := api.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var post Post
	if err = json.Unmarshal(body, &post); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	return &post, nil
}

func (api *Client) GetPostComments(service, userID, postID string) (comments []Comment, err error) {
	apiURL := fmt.Sprintf("%s/api/v1/%s/user/%s/post/%s/comments", api.BaseURL, service, userID, postID)
	resp, err := api.Get(apiURL)
	if resp != nil {
		defer func() { _ = resp.Body.Close() }()
	}
	// a post with no comments returns 404; treat that as "no comments"
	if resp != nil && resp.StatusCode == 404 {
		return comments, nil
	}
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(body, &comments); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	return comments, nil
}
