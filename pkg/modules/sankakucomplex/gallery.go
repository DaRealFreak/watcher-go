package sankakucomplex

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"
	"watcher-go/pkg/models"
)

type apiItem struct {
	Id               json.Number
	Rating           string
	Status           string
	Author           author
	SampleUrl        string `json:"sample_url"`
	SampleWidth      int    `json:"sample_width"`
	SampleHeight     int    `json:"sample_height"`
	PreviewUrl       string `json:"preview_url"`
	PreviewWidth     int    `json:"preview_width"`
	FileUrl          string `json:"file_url"`
	Width            int
	Height           int
	FileSize         int         `json:"file_size"`
	FileType         string      `json:"file_type"`
	CreatedAt        created     `json:"created_at"`
	HasChildren      bool        `json:"has_children"`
	HasComments      bool        `json:"has_comments"`
	HasNotes         bool        `json:"has_notes"`
	IsFavorite       bool        `json:"is_favorited"`
	UserVote         json.Number `json:"user_vote"`
	Md5              string
	ParentId         json.Number `json:"parent_id"`
	Change           int
	FavCount         json.Number `json:"fav_count"`
	RecommendedPosts json.Number `json:"recommended_posts"`
	RecommendedScore json.Number `json:"recommended_score"`
	VoteCount        json.Number `json:"vote_count"`
	TotalScore       json.Number `json:"total_score"`
	CommentCount     json.Number `json:"comment_count"`
	Source           string
	InVisiblePool    bool `json:"in_visible_pool"`
	IsPremium        bool `json:"is_premium"`
	Sequence         json.Number
	Tags             []tag
}

type author struct {
	Id           int
	Name         string
	Avatar       string
	AvatarRating string `json:"avatar_rating"`
}

type created struct {
	JsonClass string `json:"json_class"`
	S         int
	N         int
}

type tag struct {
	Id        int
	NameEn    string `json:"name_en"`
	NameJa    string `json:"name_ja"`
	Type      int
	Count     int
	PostCount int `json:"post_count"`
	PoolCount int `json:"pool_count"`
	Locale    string
	Rating    json.Number
	Name      string
}

// parse functionality for galleries based on the tags in the tracked item
func (m *sankakuComplex) parseGallery(item *models.TrackedItem) (downloadQueue []models.DownloadQueueItem) {
	tag := m.extractItemTag(item)
	page := 0
	foundCurrentItem := false

	for foundCurrentItem == false {
		page += 1
		apiUri := fmt.Sprintf("https://capi-v2.sankakucomplex.com/posts?lang=english&page=%d&limit=100&tags=%s", page, url.QueryEscape(tag))
		response, _ := m.get(apiUri, 0)
		apiItems := m.parseApiResponse(response)
		for _, data := range apiItems {
			if string(data.Id) != item.CurrentItem {
				downloadQueue = append(downloadQueue, models.DownloadQueueItem{
					ItemId:      string(data.Id),
					DownloadTag: path.Join(m.SanitizePath(tag, false), m.getTagSubDirectory(data)),
					FileName:    string(data.Id) + "_" + m.GetFileName(data.FileUrl),
					FileUri:     data.FileUrl,
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
func (m *sankakuComplex) parseApiResponse(response *http.Response) []apiItem {
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
