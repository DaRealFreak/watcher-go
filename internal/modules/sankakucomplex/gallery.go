package sankakucomplex

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"strconv"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
	log "github.com/sirupsen/logrus"
)

type apiResponse struct {
	Meta apiMeta    `json:"meta"`
	Data []*apiItem `json:"data"`
}

type apiMeta struct {
	Next string `json:"next"`
	Prev string `json:"prev"`
}

// apiItem is the JSON struct of item objects returned by the API
type apiItem struct {
	ID               string      `json:"id"`
	Rating           string      `json:"rating"`
	Status           string      `json:"status"`
	Author           apiAuthor   `json:"author"`
	SampleURL        string      `json:"sample_url"`
	SampleWidth      int         `json:"sample_width"`
	SampleHeight     int         `json:"sample_height"`
	PreviewURL       string      `json:"preview_url"`
	PreviewWidth     int         `json:"preview_width"`
	FileURL          string      `json:"file_url"`
	Width            int         `json:"width"`
	Height           int         `json:"height"`
	FileSize         int         `json:"file_size"`
	FileType         string      `json:"file_type"`
	CreatedAt        apiCreated  `json:"created_at"`
	HasChildren      bool        `json:"has_children"`
	HasComments      bool        `json:"has_comments"`
	HasNotes         bool        `json:"has_notes"`
	IsFavorite       bool        `json:"is_favorited"`
	InVisiblePool    bool        `json:"in_visible_pool"`
	IsPremium        bool        `json:"is_premium"`
	UserVote         json.Number `json:"user_vote"`
	Md5              string      `json:"md5"`
	ParentID         string      `json:"parent_id"`
	Change           int         `json:"change"`
	FavCount         json.Number `json:"fav_count"`
	RecommendedPosts json.Number `json:"recommended_posts"`
	RecommendedScore json.Number `json:"recommended_score"`
	VoteCount        json.Number `json:"vote_count"`
	TotalScore       json.Number `json:"total_score"`
	CommentCount     json.Number `json:"comment_count"`
	Source           string      `json:"source"`
	Sequence         json.Number `json:"sequence"`
	Tags             []apiTag    `json:"tags"`
	TagNames         []string    `json:"tag_names"`
}

type apiBookResponse struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	NameEn     string      `json:"name_en"`
	NameJa     string      `json:"name_ja"`
	PostCount  json.Number `json:"post_count"`
	PagesCount json.Number `json:"pages_count"`
}

// apiAuthor is the JSON struct of author objects returned by the API
type apiAuthor struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	DisplayName  string `json:"display_name"`
	Level        int    `json:"level"`
	Avatar       string `json:"avatar"`
	AvatarRating string `json:"avatar_rating"`
}

// apiCreated is the JSON struct of created objects returned by the API
type apiCreated struct {
	JSONClass string `json:"json_class"`
	S         int64  `json:"s"`
	N         int    `json:"n"`
}

// apiTag is the JSON struct of tag objects returned by the API
type apiTag struct {
	ID                  string `json:"id"`
	NameEn              string `json:"name_en"`
	NameJa              string `json:"name_ja"`
	Type                int    `json:"type"`
	Count               int    `json:"count"`
	PostCount           int    `json:"post_count"`
	PoolCount           int    `json:"pool_count"`
	CompanionCount      int    `json:"companion_count"`
	SeriesCount         int    `json:"series_count"`
	Locale              string `json:"locale"`
	Rating              string `json:"rating"`
	Version             *int   `json:"version"`
	TagName             string `json:"tagName"`
	TotalPostCount      int    `json:"total_post_count"`
	TotalPoolCount      int    `json:"total_pool_count"`
	IsFollowing         bool   `json:"is_following"`
	NotificationEnabled bool   `json:"notification_enabled"`
	Name                string `json:"name"`
}

// parseGallery parses galleries based on the tags in the tracked item
func (m *sankakuComplex) parseGallery(item *models.TrackedItem) (galleryItems []*downloadGalleryItem, err error) {
	originalTag, err := m.extractItemTag(item)
	if err != nil {
		return nil, err
	}

	tag := originalTag
	nextItem := ""
	foundCurrentItem := false

	for !foundCurrentItem {
		apiURI := fmt.Sprintf(
			"https://sankakuapi.com/v2/posts/keyset?lang=en&limit=100&tags=%s",
			url.QueryEscape(tag),
		)
		if nextItem != "" {
			apiURI = fmt.Sprintf("%s&next=%s", apiURI, nextItem)
		}

		response, responseErr := m.Session.Get(apiURI)
		if responseErr != nil {
			return nil, responseErr
		}

		var apiGalleryResponse apiResponse
		if err = m.parseAPIResponse(response, &apiGalleryResponse); err != nil {
			return nil, err
		}

		if nextItem == "" && len(apiGalleryResponse.Data) == 0 {
			log.WithField("module", m.Key).Warn(
				fmt.Sprintf("first request has no results, tag probably changed for uri %s", item.URI),
			)
		}

		nextItem = apiGalleryResponse.Meta.Next

		for _, data := range apiGalleryResponse.Data {
			itemTimestamp := data.CreatedAt.S

			currentItemTimestamp, _ := strconv.ParseInt(item.CurrentItem, 10, 64)
			if item.CurrentItem == "" || itemTimestamp > currentItemTimestamp {
				// direct link with ID is https://www.sankakucomplex.com/post/show/{data.ID}
				// in recent updates they used slugs, so we couldn't directly retrieve the URL based on the ID
				if data.FileURL != "" {
					galleryItems = append(galleryItems, &downloadGalleryItem{
						item: &models.DownloadQueueItem{
							ItemID: strconv.FormatInt(data.CreatedAt.S, 10),
							DownloadTag: path.Join(
								fp.TruncateMaxLength(fp.SanitizePath(m.getDownloadTag(item), false)),
								// m.getTagSubDirectory(data),
							),
							FileName: fmt.Sprintf(
								"%d_%s_%s",
								data.CreatedAt.S,
								fp.GetFileName(data.ID),
								fp.GetFileName(data.FileURL),
							),
							FileURI:         data.FileURL,
							FallbackFileURI: data.SampleURL,
						},
						apiData: data,
					})
				}
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

	// reverse queue to get the oldest "new" item first and manually update it
	for i, j := 0, len(galleryItems)-1; i < j; i, j = i+1, j-1 {
		galleryItems[i], galleryItems[j] = galleryItems[j], galleryItems[i]
	}

	return galleryItems, nil
}

// extractItemTag extracts the tag from the passed item URL
func (m *sankakuComplex) extractItemTag(item *models.TrackedItem) (string, error) {
	u, _ := url.Parse(item.URI)
	q, _ := url.ParseQuery(u.RawQuery)

	if len(q["tags"]) == 0 {
		return "", fmt.Errorf("parsed uri(%s) does not contain any \"tags\" tag", item.URI)
	}

	return q["tags"][0], nil
}

func (m *sankakuComplex) getBookResponse(postId string) (*apiBookResponse, error) {
	apiURI := fmt.Sprintf(
		"https://sankakuapi.com/post/%s/pools?lang=en",
		url.QueryEscape(postId),
	)

	response, responseErr := m.Session.Get(apiURI)
	if responseErr != nil {
		return nil, responseErr
	}

	var booksResponse []apiBookResponse
	if err := m.parseAPIResponse(response, &booksResponse); err != nil {
		return nil, err
	}

	if len(booksResponse) == 0 {
		return nil, nil
	}

	return &booksResponse[0], nil
}

// getTagSubDirectory returns possible subdirectories since the books got kinda overhand
func (m *sankakuComplex) getTagSubDirectory(item *apiItem) string {
	for _, tag := range item.Tags {
		if tag.NameEn == "doujinshi" || tag.NameEn == "comic" {
			return "book"
		}
	}

	return ""
}

func (m *sankakuComplex) getDownloadTag(item *models.TrackedItem) string {
	if item.SubFolder != "" {
		return item.SubFolder
	}

	originalTag, err := m.extractItemTag(item)
	if err != nil {
		return ""
	}

	return originalTag
}
