package schalenetwork

import (
	"encoding/json"
	"fmt"
	http "github.com/bogdanfinn/fhttp"
	"io"
	"net/url"
	"strings"
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
	ID  string `json:"id"`
	Key string `json:"key"`
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
	apiURL := fmt.Sprintf("https://api.schale.network/books/detail/%s/%s", id, key)

	res, err := m.get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get book detail: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get book detail: status code %d", res.StatusCode)
	}

	var apiRes bookDetailResponse
	if err = m.mapAPIResponse(res, &apiRes); err != nil {
		return nil, fmt.Errorf("failed to map book detail response: %w", err)
	}

	return &apiRes, nil
}

func (m *schaleNetwork) getBookData(id, key string) (*bookDataResponse, error) {
	if m.crt == "" {
		return nil, fmt.Errorf(
			"crt token is required for downloading images, " +
				"set it in config (Modules.niyaniya_moe.crt) or as a cookie",
		)
	}

	apiURL := fmt.Sprintf("https://api.schale.network/books/detail/%s/%s?crt=%s", id, key, m.crt)

	res, err := m.post(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get book data: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get book data: status code %d (crt token may be expired)", res.StatusCode)
	}

	var apiRes bookDataResponse
	if err = m.mapAPIResponse(res, &apiRes); err != nil {
		return nil, fmt.Errorf("failed to map book data response: %w", err)
	}

	return &apiRes, nil
}

func (m *schaleNetwork) getBookImages(bookID, bookKey, fmtID, fmtKey, fmtW string) (*bookImageListResponse, error) {
	apiURL := fmt.Sprintf(
		"https://api.schale.network/books/data/%s/%s/%s/%s/%s?crt=%s",
		bookID, bookKey, fmtID, fmtKey, fmtW, m.crt,
	)

	res, err := m.get(apiURL)
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

func (m *schaleNetwork) getSearch(query string, page int) (*searchResponse, error) {
	apiURL := fmt.Sprintf(
		"https://api.schale.network/books?s=%s&page=%d",
		url.QueryEscape(query), page,
	)

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
