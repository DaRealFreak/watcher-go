package api

import "encoding/json"

// Envelope is the outer JSON envelope tapas wraps all responses in.
type Envelope[T any] struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Type string `json:"type"`
	Data T      `json:"data"`
}

// Pagination is the cursor/page state returned for paginated endpoints.
type Pagination struct {
	Page    int  `json:"page"`
	HasNext bool `json:"has_next"`
	Sort    string `json:"sort"`
	Limit   int  `json:"limit"`
	Total   int  `json:"total"`
}

// EpisodeListData is the data payload for the series episodes endpoint. The
// episode list itself comes back as an HTML fragment we have to scrape.
type EpisodeListData struct {
	Pagination Pagination `json:"pagination"`
	Body       string     `json:"body"`
}

// EpisodeListItem represents one entry extracted from the EpisodeListData.Body
// HTML fragment.
type EpisodeListItem struct {
	ID    string
	Title string
}

// Episode is the metadata block returned for a single episode.
type Episode struct {
	ID            json.Number `json:"id"`
	Title         string      `json:"title"`
	Scene         int         `json:"scene"`
	ThumbURL      string      `json:"thumb_url"`
	PublishDate   string      `json:"publish_date"`
	Free          bool        `json:"free"`
	Mature        bool        `json:"mature"`
	NSFW          bool        `json:"nsfw"`
	MustPay       bool        `json:"must_pay"`
	Scheduled     bool        `json:"scheduled"`
	Unlocked      bool        `json:"unlocked"`
	PrevEpisodeID json.Number `json:"prev_ep_id"`
	NextEpisodeID json.Number `json:"next_ep_id"`
}

// EpisodeData is the data payload for the single episode endpoint.
type EpisodeData struct {
	Episode Episode `json:"episode"`
	HTML    string  `json:"html"`
}
