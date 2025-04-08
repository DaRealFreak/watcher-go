package sankakucomplex

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
)

type bookApiResponse struct {
	Pools  poolsResponse  `json:"pools"`
	Series seriesResponse `json:"series"`
	Page   int            `json:"page"`
}

type poolsResponse struct {
	Meta apiMeta       `json:"meta"`
	Data []bookApiItem `json:"data"`
}

// apiItem is the JSON struct of item objects returned by the API
type bookApiItem struct {
	ID               string     `json:"id"`
	NameEn           *string    `json:"name_en"`
	NameJa           *string    `json:"name_ja"`
	Description      string     `json:"description"`
	DescriptionEn    *string    `json:"description_en"`
	DescriptionJa    *string    `json:"description_ja"`
	CreatedAt        string     `json:"created_at"`
	UpdatedAt        string     `json:"updated_at"`
	Author           apiAuthor  `json:"author"`
	Status           string     `json:"status"`
	PostCount        int        `json:"post_count"`
	PagesCount       int        `json:"pages_count"`
	VisiblePostCount int        `json:"visible_post_count"`
	Tags             []*apiTag  `json:"tags"`
	PostTags         []*apiTag  `json:"post_tags"`
	ArtistTags       []*apiTag  `json:"artist_tags"`
	GenreTags        []*apiTag  `json:"genre_tags"`
	Posts            []*apiItem `json:"posts"`
	FileURL          *string    `json:"file_url"`
	SampleURL        *string    `json:"sample_url"`
	PreviewURL       *string    `json:"preview_url"`
	IsPremium        bool       `json:"is_premium"`
	IsAnonymous      bool       `json:"is_anonymous"`
	RedirectToSignup bool       `json:"redirect_to_signup"`
	Locale           string     `json:"locale"`
	IsPublic         bool       `json:"is_public"`
	IsIntact         bool       `json:"is_intact"`
	IsRaw            bool       `json:"is_raw"`
	IsTrial          bool       `json:"is_trial"`
	IsPending        bool       `json:"is_pending"`
	IsActive         bool       `json:"is_active"`
	IsFlagged        bool       `json:"is_flagged"`
	IsDeleted        bool       `json:"is_deleted"`
	Name             string     `json:"name"`
}

type seriesResponse struct {
	Data       []series `json:"data"`
	TotalCount int      `json:"totalCount"`
}

type series struct {
}

// parseAPIBookResponse parses the book response from the API
func (m *sankakuComplex) parseAPIResponse(response *http.Response, apiRes interface{}) error {
	body, _ := io.ReadAll(response.Body)

	err := json.Unmarshal(body, &apiRes)
	if err != nil {
		return err
	}

	return err
}

func (m *sankakuComplex) extractBookItems(data bookApiItem) (downloadQueue []*downloadGalleryItem, err error) {
	tmpItem := &models.TrackedItem{
		URI: fmt.Sprintf(
			"https://www.sankakucomplex.com/?tags=%s",
			url.QueryEscape(fmt.Sprintf("pool:%s", data.ID)),
		),
	}
	tmpDownloadQueue, err := m.parseGallery(tmpItem)
	if err != nil {
		return downloadQueue, err
	}

	// reverse download queue to download old items first
	for i, j := 0, len(tmpDownloadQueue)-1; i < j; i, j = i+1, j-1 {
		tmpDownloadQueue[i], tmpDownloadQueue[j] = tmpDownloadQueue[j], tmpDownloadQueue[i]
	}

	return tmpDownloadQueue, nil
}

func (m *sankakuComplex) extractBookName(bookResponse bookApiItem) (bookTag string) {
	if bookResponse.NameEn != nil {
		bookTag = *bookResponse.NameEn
	} else {
		if bookResponse.NameJa != nil {
			bookTag = *bookResponse.NameJa
		}
	}

	return bookTag
}

func (m *sankakuComplex) extractLanguage(apiResponse bookApiItem) (languageTags []string) {
	// Mapping from the server key to the original display text.
	originalLanguages := map[string]string{
		"arabic_language":                        "Arabic",
		"english_language":                       "English",
		"japanese_language":                      "日本語",
		"bulgarian_language":                     "Български",
		"danish_language":                        "Dansk",
		"german_language":                        "Deutsch",
		"greek_language":                         "Ελληνικά",
		"spanish_language":                       "Español",
		"finnish_language":                       "Suomi",
		"french_language":                        "Français",
		"hindi_language":                         "हिन्दी",
		"hungarian_language":                     "Magyar",
		"indonesian_language":                    "Bahasa Indonesia",
		"italian_language":                       "Italiano",
		"malay_language":                         "Bahasa Melayu",
		"dutch_language":                         "Nederlands",
		"norwegian_language":                     "Norsk",
		"polish_language":                        "Polski",
		"portuguese_language":                    "Português",
		"korean_language":                        "한국어",
		"romanian_language":                      "Română",
		"russian_language":                       "Русский",
		"vatican_language":                       "Vatican",
		"swedish_language":                       "Svenska",
		"thai_language":                          "ไทย",
		"turkish_language":                       "Türkçe",
		"chinese_language":                       "中文",
		"traditional_chinese_hong_kong_language": "正體字（香港）",
		"traditional_chinese_taiwan_language":    "正體字（台湾）",
		"vietnamese_language":                    "Tiếng Việt",
	}

	// Mapping from the same server keys to their plain English names.
	englishNames := map[string]string{
		"arabic_language":                        "Arabic",
		"english_language":                       "English",
		"japanese_language":                      "Japanese",
		"bulgarian_language":                     "Bulgarian",
		"danish_language":                        "Danish",
		"german_language":                        "German",
		"greek_language":                         "Greek",
		"spanish_language":                       "Spanish",
		"finnish_language":                       "Finnish",
		"french_language":                        "French",
		"hindi_language":                         "Hindi",
		"hungarian_language":                     "Hungarian",
		"indonesian_language":                    "Indonesian",
		"italian_language":                       "Italian",
		"malay_language":                         "Malay",
		"dutch_language":                         "Dutch",
		"norwegian_language":                     "Norwegian",
		"polish_language":                        "Polish",
		"portuguese_language":                    "Portuguese",
		"korean_language":                        "Korean",
		"romanian_language":                      "Romanian",
		"russian_language":                       "Russian",
		"vatican_language":                       "Vatican",
		"swedish_language":                       "Swedish",
		"thai_language":                          "Thai",
		"turkish_language":                       "Turkish",
		"chinese_language":                       "Chinese",
		"traditional_chinese_hong_kong_language": "Traditional Chinese (Hong Kong)",
		"traditional_chinese_taiwan_language":    "Traditional Chinese (Taiwan)",
		"vietnamese_language":                    "Vietnamese",
	}

	// Create a combined lookup map that stores keys in all lower-case.
	lookup := make(map[string]string)
	for serverKey, original := range originalLanguages {
		lookup[strings.ToLower(serverKey)] = original
		if eng, ok := englishNames[serverKey]; ok {
			lookup[strings.ToLower(eng)] = original
		}
	}

	// Helper function that does case-insensitive lookup.
	getOriginalDisplay := func(key string) string {
		return lookup[strings.ToLower(key)]
	}

	for _, tag := range apiResponse.Tags {
		if getOriginalDisplay(tag.TagName) != "" {
			languageTags = append(languageTags, getOriginalDisplay(tag.TagName))
		}
	}

	return languageTags
}

// parseBooks parses tracked items identified as books
// nolint: unused
func (m *sankakuComplex) parseBooks(item *models.TrackedItem) (downloadBookItems []*downloadBookItem, err error) {
	originalTag, err := m.extractItemTag(item)
	if err != nil {
		return nil, err
	}

	bookCreatedAtLayout := "2006-01-02 15:04"

	tag := originalTag
	page := 1
	foundCurrentItem := false

	for !foundCurrentItem {
		apiURI := fmt.Sprintf(
			"https://sankakuapi.com/poolseriesv2?lang=en&filledPools=true&offset=0&limit=100&tags=order:date+%s&page=%d&includes[]=pools&exceptStatuses[]=deleted",
			url.QueryEscape(tag),
			page,
		)

		response, err := m.Session.Get(apiURI)
		if err != nil {
			return nil, err
		}

		var apiGalleryResponse bookApiResponse
		if err = m.parseAPIResponse(response, &apiGalleryResponse); err != nil {
			return nil, err
		}

		for _, data := range apiGalleryResponse.Pools.Data {
			itemTime, timeErr := time.Parse(bookCreatedAtLayout, data.CreatedAt)
			if timeErr != nil {
				return nil, fmt.Errorf("error parsing book created at time: %w (%s)",
					timeErr,
					data.CreatedAt,
				)
			}

			currentItemTimestamp, _ := strconv.ParseInt(item.CurrentItem, 10, 64)
			if item.CurrentItem == "" || itemTime.Unix() > currentItemTimestamp {
				bookTag := m.extractBookName(data)
				if bookTag == "" {
					return nil, fmt.Errorf("no book tag could be extracted for book with id: %s", data.ID)
				}

				downloadBookItems = append(downloadBookItems, &downloadBookItem{
					bookId:       strconv.FormatInt(itemTime.Unix(), 10),
					bookName:     bookTag,
					bookLanguage: m.extractLanguage(data),
					bookApiItem:  data,
				})
			} else {
				foundCurrentItem = true
				break
			}
		}

		// we reached the last possible page, break here
		if apiGalleryResponse.Pools.Meta.Next == "" {
			break
		}

		page++
	}

	// reverse download queue to download old items first
	for i, j := 0, len(downloadBookItems)-1; i < j; i, j = i+1, j-1 {
		downloadBookItems[i], downloadBookItems[j] = downloadBookItems[j], downloadBookItems[i]
	}

	return downloadBookItems, nil
}

func (m *sankakuComplex) parseSingleBook(item *models.TrackedItem, bookId string) (galleryItems []*downloadGalleryItem, err error) {
	apiURI := fmt.Sprintf(
		"https://sankakuapi.com/pools/%s?lang=en&includes[]=series",
		url.QueryEscape(bookId),
	)

	response, err := m.Session.Get(apiURI)
	if err != nil {
		return nil, err
	}

	var bookResponse bookApiItem
	if err = m.parseAPIResponse(response, &bookResponse); err != nil {
		return nil, err
	}

	bookTag := m.extractBookName(bookResponse)
	if bookTag == "" {
		return nil, fmt.Errorf("no book tag could be extracted for book with id: %s", bookResponse.ID)
	}

	bookLanguages := m.extractLanguage(bookResponse)
	bookLanguage := ""
	if len(bookLanguages) > 0 {
		bookLanguage = fmt.Sprintf(" [%s]", strings.Join(bookLanguages, ", "))
	}

	for i, galleryItem := range bookResponse.Posts {
		itemTimestamp := galleryItem.CreatedAt.S

		currentItemTimestamp, _ := strconv.ParseInt(item.CurrentItem, 10, 64)
		if item.CurrentItem == "" || itemTimestamp > currentItemTimestamp {
			if galleryItem.FileURL != "" {
				galleryItems = append(galleryItems, &downloadGalleryItem{
					item: &models.DownloadQueueItem{
						ItemID:          strconv.FormatInt(galleryItem.CreatedAt.S, 10),
						DownloadTag:     fmt.Sprintf("%s/%s%s (%s)", "books", fp.SanitizePath(bookTag, false), bookLanguage, bookId),
						FileName:        fmt.Sprintf("%d_%s", i+1, fp.GetFileName(galleryItem.FileURL)),
						FileURI:         galleryItem.FileURL,
						FallbackFileURI: galleryItem.SampleURL,
					},
					apiData: galleryItem,
				})
			}
		}
	}

	return galleryItems, nil
}
