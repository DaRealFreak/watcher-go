package api

import (
	"encoding/json"
	"fmt"
	"net/url"
)

type ApiResponse struct {
	Meta apiMeta    `json:"meta"`
	Data []*ApiItem `json:"data"`
}

type apiMeta struct {
	Next string `json:"next"`
	Prev string `json:"prev"`
}

// ApiItem is the JSON struct of item objects returned by the API
type ApiItem struct {
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
	Tags             []ApiTag    `json:"tags"`
	TagNames         []string    `json:"tag_names"`
}

type ApiBookResponse struct {
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

// ApiTag is the JSON struct of tag objects returned by the API
type ApiTag struct {
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

func (a *SankakuComplexApi) GetPosts(tag string, nextItem string) (*ApiResponse, error) {
	apiURI := fmt.Sprintf(
		"https://sankakuapi.com/v2/posts/keyset?lang=en&limit=100&tags=%s",
		url.QueryEscape(tag),
	)
	if nextItem != "" {
		apiURI = fmt.Sprintf("%s&next=%s", apiURI, nextItem)
	}

	response, responseErr := a.get(apiURI)
	if responseErr != nil {
		return nil, responseErr
	}

	var apiRes ApiResponse
	if err := a.parseAPIResponse(response, &apiRes); err != nil {
		return nil, err
	}

	return &apiRes, nil
}
