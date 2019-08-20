package sankakucomplex

import (
	"encoding/json"
	"fmt"
	"github.com/DaRealFreak/watcher-go/pkg/models"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
)

type apiItem struct {
	ID               json.Number `json:"id"`
	Rating           string      `json:"rating"`
	Status           string      `json:"status"`
	Author           author      `json:"author"`
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
	CreatedAt        created     `json:"created_at"`
	HasChildren      bool        `json:"has_children"`
	HasComments      bool        `json:"has_comments"`
	HasNotes         bool        `json:"has_notes"`
	IsFavorite       bool        `json:"is_favorited"`
	InVisiblePool    bool        `json:"in_visible_pool"`
	IsPremium        bool        `json:"is_premium"`
	UserVote         json.Number `json:"user_vote"`
	Md5              string      `json:"md5"`
	ParentID         json.Number `json:"parent_id"`
	Change           int         `json:"change"`
	FavCount         json.Number `json:"fav_count"`
	RecommendedPosts json.Number `json:"recommended_posts"`
	RecommendedScore json.Number `json:"recommended_score"`
	VoteCount        json.Number `json:"vote_count"`
	TotalScore       json.Number `json:"total_score"`
	CommentCount     json.Number `json:"comment_count"`
	Source           string      `json:"source"`
	Sequence         json.Number `json:"sequence"`
	Tags             []tag       `json:"tags"`
}

type author struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Avatar       string `json:"avatar"`
	AvatarRating string `json:"avatar_rating"`
}

type created struct {
	JSONClass string `json:"json_class"`
	S         int    `json:"s"`
	N         int    `json:"n"`
}

type tag struct {
	ID        int         `json:"id"`
	NameEn    string      `json:"name_en"`
	NameJa    string      `json:"name_ja"`
	Type      int         `json:"type"`
	Count     int         `json:"count"`
	PostCount int         `json:"post_count"`
	PoolCount int         `json:"pool_count"`
	Locale    string      `json:"locale"`
	Rating    json.Number `json:"rating"`
	Name      string      `json:"name"`
}

// parse functionality for galleries based on the tags in the tracked item
func (m *sankakuComplex) parseGallery(item *models.TrackedItem) (downloadQueue []models.DownloadQueueItem) {
	tag := m.extractItemTag(item)
	page := 0
	foundCurrentItem := false

	for !foundCurrentItem {
		page++
		apiURI := fmt.Sprintf(
			"https://capi-v2.sankakucomplex.com/posts?lang=english&page=%d&limit=100&tags=%s",
			page,
			url.QueryEscape(tag),
		)
		response, _ := m.Session.Get(apiURI)
		apiItems := m.parseAPIResponse(response)
		for _, data := range apiItems {
			if string(data.ID) > item.CurrentItem || item.CurrentItem == "" {
				downloadQueue = append(downloadQueue, models.DownloadQueueItem{
					ItemId:      string(data.ID),
					DownloadTag: path.Join(m.SanitizePath(tag, false), m.getTagSubDirectory(data)),
					FileName:    string(data.ID) + "_" + m.GetFileName(data.FileURL),
					FileUri:     data.FileURL,
				})
			} else {
				foundCurrentItem = true
				break
			}
		}

		// we reached the last possible page, break here
		if len(apiItems) != 100 {
			break
		}
	}

	// reverse queue to get the oldest "new" item first and manually update it
	downloadQueue = m.ReverseDownloadQueueItems(downloadQueue)

	return downloadQueue
}

// parse the response from the API
func (m *sankakuComplex) parseAPIResponse(response *http.Response) []apiItem {
	body, _ := ioutil.ReadAll(response.Body)
	var apiItems []apiItem
	_ = json.Unmarshal(body, &apiItems)
	return apiItems
}

// extract the tag from the passed item to use in the API request
func (m *sankakuComplex) extractItemTag(item *models.TrackedItem) string {
	u, _ := url.Parse(item.Uri)
	q, _ := url.ParseQuery(u.RawQuery)
	if len(q["tags"]) == 0 {
		log.Fatalf("parsed uri(%s) does not contain any \"tags\" tag", item.Uri)
	}
	return q["tags"][0]
}

// since the books got kinda overhand, sort some items in sub folders based on the tags
func (m *sankakuComplex) getTagSubDirectory(item apiItem) string {
	for _, tag := range item.Tags {
		if tag.NameEn == "doujinshi" {
			return "book"
		}
	}
	return ""
}
