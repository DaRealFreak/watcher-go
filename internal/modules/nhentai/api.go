package nhentai

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"

	http "github.com/bogdanfinn/fhttp"
)

const (
	SortingDate         = "date"
	SortingPopularToday = "popular-today"
	SortingPopularWeek  = "popular-week"
	SortingPopular      = "popular"
	SortingPopularMonth = "popular-month"
)

type galleryPage struct {
	Number          int    `json:"number"`
	Path            string `json:"path"`
	Width           int    `json:"width"`
	Height          int    `json:"height"`
	Thumbnail       string `json:"thumbnail"`
	ThumbnailWidth  int    `json:"thumbnail_width"`
	ThumbnailHeight int    `json:"thumbnail_height"`
}

type galleryTag struct {
	ID    json.Number `json:"id"`
	Type  string      `json:"type"`
	Name  string      `json:"name"`
	Slug  string      `json:"slug"`
	URL   string      `json:"url"`
	Count int         `json:"count"`
}

type galleryResponse struct {
	GalleryID json.Number `json:"id"`
	MediaID   json.Number `json:"media_id"`
	Title     struct {
		English  string `json:"english"`
		Japanese string `json:"japanese"`
		Pretty   string `json:"pretty"`
	} `json:"title"`
	Cover struct {
		Path   string `json:"path"`
		Width  int    `json:"width"`
		Height int    `json:"height"`
	} `json:"cover"`
	Scanlator    string        `json:"scanlator"`
	UploadDate   int64         `json:"upload_date"`
	Tags         []galleryTag  `json:"tags"`
	NumPages     int           `json:"num_pages"`
	NumFavorites int           `json:"num_favorites"`
	Pages        []galleryPage `json:"pages"`
}

type searchResultItem struct {
	ID              json.Number `json:"id"`
	MediaID         json.Number `json:"media_id"`
	EnglishTitle    string      `json:"english_title"`
	JapaneseTitle   string      `json:"japanese_title"`
	Thumbnail       string      `json:"thumbnail"`
	ThumbnailWidth  int         `json:"thumbnail_width"`
	ThumbnailHeight int         `json:"thumbnail_height"`
	NumPages        int         `json:"num_pages"`
	TagIDs          []int       `json:"tag_ids"`
	Blacklisted     bool        `json:"blacklisted"`
}

type searchResponse struct {
	Result   []*searchResultItem `json:"result"`
	NumPages json.Number         `json:"num_pages"`
	PerPage  json.Number         `json:"per_page"`
}

type tagResponse struct {
	ID    json.Number `json:"id"`
	Type  string      `json:"type"`
	Name  string      `json:"name"`
	Slug  string      `json:"slug"`
	URL   string      `json:"url"`
	Count int         `json:"count"`
}

func (a *galleryResponse) GetLanguage() string {
	textlessTag := "[Textless]"
	if strings.Contains(a.Title.Japanese, textlessTag) || strings.Contains(a.Title.English, textlessTag) {
		// if the title contains the textless tag, we return Textless as language
		return "Textless"
	}

	if len(a.Tags) == 0 {
		return ""
	}

	for _, tag := range a.Tags {
		if tag.Type == "language" {
			// skip the translated tag
			if tag.Name == "translated" {
				continue
			}

			// capitalize the first letter of the language name
			if len(tag.Name) > 0 {
				return strings.ToUpper(tag.Name[:1]) + tag.Name[1:]
			}
		}
	}

	return ""
}

func (a *galleryResponse) GetImages() []string {
	images := make([]string, 0, len(a.Pages))

	for _, page := range a.Pages {
		imageURL := fmt.Sprintf("https://i.nhentai.net/%s", page.Path)
		images = append(images, imageURL)
	}

	return images
}

func (a *galleryResponse) GetTitle() string {
	galleryTitle := a.Title.Pretty
	if galleryTitle == "" {
		galleryTitle = a.Title.English
		if galleryTitle == "" {
			galleryTitle = a.Title.Japanese
		}
	}

	return galleryTitle
}

func (a *galleryResponse) GetURL() string {
	return fmt.Sprintf("https://nhentai.net/g/%s/", a.GalleryID)
}

func (s *searchResultItem) GetURL() string {
	return fmt.Sprintf("https://nhentai.net/g/%s/", s.ID)
}

func (m *nhentai) getGallery(galleryID string) (*galleryResponse, error) {
	// construct the request URL
	apiUrl := fmt.Sprintf("https://nhentai.net/api/v2/galleries/%s", galleryID)

	// make the request
	res, err := m.get(apiUrl)
	if err != nil || res == nil {
		if res != nil && res.StatusCode == 503 {
			return nil, fmt.Errorf(
				"returned status code was 503, check cloudflare.user_agent setting and cf_clearance cookie." +
					"cloudflare checks used IP and User-Agent to validate the cf_clearance cookie",
			)
		}

		return nil, fmt.Errorf("failed to get gallery: %w", err)
	}

	// check if the response is successful
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get gallery: status code %d", res.StatusCode)
	}

	// map the response into the galleryResponse struct
	var apiRes galleryResponse
	if err = m.mapAPIResponse(res, &apiRes); err != nil {
		return nil, fmt.Errorf("failed to map API response: %w", err)
	}

	return &apiRes, nil
}

func (m *nhentai) getSearch(searchQuery string, page int, sorting string) (*searchResponse, error) {
	params := url.Values{
		"query": []string{searchQuery},
		"page":  []string{strconv.Itoa(page)},
		"sort":  []string{sorting},
	}

	apiUrl := fmt.Sprintf("https://nhentai.net/api/v2/search?%s", params.Encode())
	res, err := m.get(apiUrl)
	if err != nil || res == nil {
		if res != nil && res.StatusCode == 503 {
			return nil, fmt.Errorf(
				"returned status code was 503, check cloudflare.user_agent setting and cf_clearance cookie." +
					"cloudflare checks used IP and User-Agent to validate the cf_clearance cookie",
			)
		}

		return nil, fmt.Errorf("failed to get search results: %w", err)
	}

	// check if the response is successful
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get search results: status code %d", res.StatusCode)
	}

	// map the response into the searchResponse struct
	var apiRes searchResponse
	if err = m.mapAPIResponse(res, &apiRes); err != nil {
		return nil, fmt.Errorf("failed to map API response: %w", err)
	}

	return &apiRes, nil
}

func (m *nhentai) getTag(tagType string, slug string) (*tagResponse, error) {
	apiUrl := fmt.Sprintf("https://nhentai.net/api/v2/tags/%s/%s", tagType, slug)
	res, err := m.get(apiUrl)
	if err != nil || res == nil {
		if res != nil && res.StatusCode == 503 {
			return nil, fmt.Errorf(
				"returned status code was 503, check cloudflare.user_agent setting and cf_clearance cookie." +
					"cloudflare checks used IP and User-Agent to validate the cf_clearance cookie",
			)
		}

		return nil, fmt.Errorf("failed to get tag: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get tag: status code %d", res.StatusCode)
	}

	var apiRes tagResponse
	if err = m.mapAPIResponse(res, &apiRes); err != nil {
		return nil, fmt.Errorf("failed to map API response: %w", err)
	}

	return &apiRes, nil
}

func (m *nhentai) getTaggedGalleries(tagID string, page int, sorting string) (*searchResponse, error) {
	params := url.Values{
		"tag_id": []string{tagID},
		"sort":   []string{sorting},
		"page":   []string{strconv.Itoa(page)},
	}

	apiUrl := fmt.Sprintf("https://nhentai.net/api/v2/galleries/tagged?%s", params.Encode())
	res, err := m.get(apiUrl)
	if err != nil || res == nil {
		if res != nil && res.StatusCode == 503 {
			return nil, fmt.Errorf(
				"returned status code was 503, check cloudflare.user_agent setting and cf_clearance cookie." +
					"cloudflare checks used IP and User-Agent to validate the cf_clearance cookie",
			)
		}

		return nil, fmt.Errorf("failed to get tagged galleries: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get tagged galleries: status code %d", res.StatusCode)
	}

	var apiRes searchResponse
	if err = m.mapAPIResponse(res, &apiRes); err != nil {
		return nil, fmt.Errorf("failed to map API response: %w", err)
	}

	return &apiRes, nil
}

// mapAPIResponse maps the API response into the passed APIResponse type
func (m *nhentai) mapAPIResponse(res *http.Response, apiRes interface{}) error {
	out, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	content := string(out)

	if res.StatusCode >= 400 {
		return fmt.Errorf(`unknown error response: "%s"`, content)
	}

	// unmarshal the request content into the response struct
	if err = json.Unmarshal([]byte(content), &apiRes); err != nil {
		return err
	}

	return nil
}
