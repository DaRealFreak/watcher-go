package sankakucomplex

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
)

const LanguageMetaType = 8

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
	languageTagPattern := regexp.MustCompile("(.*)_language")
	for _, tag := range apiResponse.Tags {
		if tag.Type == LanguageMetaType {
			if languageTagPattern.MatchString(tag.Name) {
				languageTags = append(languageTags, languageTagPattern.FindStringSubmatch(tag.Name)[1])
			}
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
