package pixiv

import (
	"encoding/json"
	"time"
)

// publicPagination is the struct for the previous API pagination response object
type publicPagination struct {
	Previous json.Number `json:"previous"`
	Next     json.Number `json:"next"`
	Current  json.Number `json:"current"`
	PerPage  json.Number `json:"per_page"`
	Total    json.Number `json:"total"`
	Pages    json.Number `json:"pages"`
}

// publicSearchResponse is the struct for the previous API search response object
type publicSearchResponse struct {
	Status        string                `json:"status"`
	Illustrations []*publicIllustration `json:"response"`
	Count         json.Number           `json:"count"`
	Pagination    *publicPagination     `json:"pagination"`
}

// publicIllustration is the struct for the previous API illustration response object
type publicIllustration struct {
	ID             json.Number                    `json:"id"`
	Title          string                         `json:"title"`
	Type           string                         `json:"type"`
	ImageUrls      map[string]string              `json:"image_urls"`
	Caption        string                         `json:"caption"`
	Restrict       json.Number                    `json:"restrict"`
	User           *user                          `json:"user"`
	Tags           []string                       `json:"tags"`
	Tools          []string                       `json:"tools"`
	CreateDate     time.Time                      `json:"create_date"`
	PageCount      json.Number                    `json:"page_count"`
	Width          json.Number                    `json:"width"`
	Height         json.Number                    `json:"height"`
	SanityLevel    json.Number                    `json:"sanity_level"`
	XRestrict      json.Number                    `json:"x_restrict"`
	Series         *series                        `json:"series"`
	MetaSinglePage map[string]string              `json:"meta_single_page"`
	MetaPages      []map[string]map[string]string `json:"meta_pages"`
	TotalView      json.Number                    `json:"total_view"`
	TotalBookmarks json.Number                    `json:"total_bookmarks"`
	IsBookmarked   bool                           `json:"is_bookmarked"`
	Visible        bool                           `json:"visible"`
	IsMuted        bool                           `json:"is_muted"`
	TotalComments  json.Number                    `json:"total_comments"`
}
