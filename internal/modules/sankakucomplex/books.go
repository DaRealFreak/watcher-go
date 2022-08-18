package sankakucomplex

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
)

const LanguageMetaType = 8

type bookApiResponse struct {
	Meta apiMeta       `json:"meta"`
	Data []bookApiItem `json:"data"`
}

// apiItem is the JSON struct of item objects returned by the API
type bookApiItem struct {
	ID     json.Number `json:"id"`
	NameEn *string     `json:"name_en"`
	NameJa *string     `json:"name_ja"`
	Author apiAuthor   `json:"author"`
	Tags   []*apiTag   `json:"tags"`
	Posts  []*apiItem  `json:"posts"`
}

// parseAPIBookResponse parses the book response from the API
func (m *sankakuComplex) parseAPIResponse(response *http.Response, apiRes interface{}) error {
	body, _ := ioutil.ReadAll(response.Body)

	err := json.Unmarshal(body, &apiRes)
	if err != nil {
		return err
	}

	return err
}

func (m *sankakuComplex) extractBookItems(data bookApiItem) (downloadQueue []*downloadGalleryItem, err error) {
	tmpItem := &models.TrackedItem{
		URI: fmt.Sprintf(
			"https://beta.sankakucomplex.com/?tags=%s",
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

	tag := originalTag
	nextItem := ""
	foundCurrentItem := false

	for !foundCurrentItem {
		apiURI := fmt.Sprintf(
			"https://capi-v2.sankakucomplex.com/pools/keyset?lang=en&limit=100&includes[]=series&tags=%s&pool_type=0",
			url.QueryEscape(tag),
		)
		if nextItem != "" {
			apiURI = fmt.Sprintf("%s&next=%s", apiURI, nextItem)
		}

		response, err := m.Session.Get(apiURI)
		if err != nil {
			return nil, err
		}

		var apiGalleryResponse bookApiResponse
		if err = m.parseAPIResponse(response, &apiGalleryResponse); err != nil {
			return nil, err
		}

		nextItem = apiGalleryResponse.Meta.Next

		for _, data := range apiGalleryResponse.Data {
			itemID, err := data.ID.Int64()
			if err != nil {
				return nil, err
			}

			// will return 0 on error, so fine for us too
			currentItemID, _ := strconv.ParseInt(item.CurrentItem, 10, 64)
			if item.CurrentItem == "" || itemID > currentItemID {
				tmpDownloadQueue, err := m.extractBookItems(data)
				if err != nil {
					return downloadBookItems, err
				}

				bookTag := m.extractBookName(data)
				if bookTag == "" {
					return nil, fmt.Errorf("no book tag could be extracted for book with id: %s", data.ID)
				}

				downloadBookItems = append(downloadBookItems, &downloadBookItem{
					bookId:       data.ID.String(),
					bookName:     bookTag,
					bookLanguage: m.extractLanguage(data),
					items:        tmpDownloadQueue,
				})
			} else {
				foundCurrentItem = true
				break
			}
		}

		// we reached the last possible page, break here
		if len(apiGalleryResponse.Data) == 0 || nextItem == "" {
			break
		}
	}

	// reverse download queue to download old items first
	for i, j := 0, len(downloadBookItems)-1; i < j; i, j = i+1, j-1 {
		downloadBookItems[i], downloadBookItems[j] = downloadBookItems[j], downloadBookItems[i]
	}

	return downloadBookItems, nil
}

func (m *sankakuComplex) parseSingleBook(item *models.TrackedItem, bookId string) (galleryItems []*downloadGalleryItem, err error) {
	apiURI := fmt.Sprintf(
		"https://capi-v2.sankakucomplex.com/pools/%s?lang=en&includes[]=series",
		url.QueryEscape(bookId),
	)

	response, err := m.Session.Get(apiURI)
	if err != nil {
		return nil, err
	}

	var apiBookResponse bookApiItem
	if err = m.parseAPIResponse(response, &apiBookResponse); err != nil {
		return nil, err
	}

	bookTag := m.extractBookName(apiBookResponse)
	if bookTag == "" {
		return nil, fmt.Errorf("no book tag could be extracted for book with id: %s", apiBookResponse.ID)
	}

	bookLanguages := m.extractLanguage(apiBookResponse)
	bookLanguage := ""
	if len(bookLanguages) > 0 {
		bookLanguage = fmt.Sprintf(" [%s]", strings.Join(bookLanguages, ", "))
	}

	for i, galleryItem := range apiBookResponse.Posts {
		itemID, err := galleryItem.ID.Int64()
		if err != nil {
			return nil, err
		}

		// will return 0 on error, so fine for us too
		currentItemID, _ := strconv.ParseInt(item.CurrentItem, 10, 64)
		if item.CurrentItem == "" || itemID > currentItemID {
			if galleryItem.FileURL != "" {
				galleryItems = append(galleryItems, &downloadGalleryItem{
					item: &models.DownloadQueueItem{
						ItemID:          string(galleryItem.ID),
						DownloadTag:     fmt.Sprintf("%s/%s%s (%s)", "books", fp.SanitizePath(bookTag, false), bookLanguage, bookId),
						FileName:        fmt.Sprintf("%d_%s", i+1, fp.GetFileName(galleryItem.FileURL)),
						FileURI:         galleryItem.FileURL,
						FallbackFileURI: galleryItem.SampleURL,
					},
				})
			}
		}
	}

	return galleryItems, nil
}
