package nhentai

import (
	"encoding/json"
	"fmt"
	http "github.com/bogdanfinn/fhttp"
	"io"
	"net/url"
	"strconv"
	"strings"
)

const (
	SortingRecent       = "recent"
	SortingPopularToday = "popular-today"
	SortingPopularWeek  = "popular-week"
	SortingPopular      = "popular"
	SortingPopularMonth = "popular-month"
)

type galleryResponse struct {
	GalleryID json.Number `json:"id"`
	MediaID   json.Number `json:"media_id"`
	Title     struct {
		English  string `json:"english"`
		Japanese string `json:"japanese"`
		Pretty   string `json:"pretty"`
	} `json:"title"`
	Images struct {
		Pages []struct {
			Type   string `json:"t"`
			Width  int    `json:"w"`
			Height int    `json:"h"`
		} `json:"pages"`
	} `json:"images"`
	Tags []struct {
		ID   json.Number `json:"id"`
		Type string      `json:"type"`
		Name string      `json:"name"`
		URL  string      `json:"url"`
	} `json:"tags"`
}

type searchResponse struct {
	Galleries []*galleryResponse `json:"result"`
	NumPages  json.Number        `json:"num_pages"`
	PerPage   json.Number        `json:"per_page"`
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
	images := make([]string, 0, len(a.Images.Pages))

	for i, page := range a.Images.Pages {
		// page.Type == "j" case
		imageExtension := ""
		switch page.Type {
		case "j":
			imageExtension = "jpg"
		case "p":
			imageExtension = "png"
		case "w":
			imageExtension = "webp"
		}

		imageURL := fmt.Sprintf("https://i.nhentai.net/galleries/%s/%d.%s", a.MediaID, i+1, imageExtension)
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

func (m *nhentai) getGallery(galleryID string) (*galleryResponse, error) {
	// construct the request URL
	apiUrl := fmt.Sprintf("https://nhentai.net/api/gallery/%s", galleryID)

	// make the request
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

	apiUrl := fmt.Sprintf("https://nhentai.net/api/galleries/search?%s", params.Encode())
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

	// map the response into the galleryResponse struct
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
