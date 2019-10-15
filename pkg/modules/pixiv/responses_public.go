package pixiv

import (
	"encoding/json"
)

type publicPagination struct {
	Previous json.Number `json:"previous"`
	Next     json.Number `json:"next"`
	Current  json.Number `json:"current"`
	PerPage  json.Number `json:"per_page"`
	Total    json.Number `json:"total"`
	Pages    json.Number `json:"pages"`
}

type publicSearchResponse struct {
	Status        string            `json:"status"`
	Illustrations []*illustration   `json:"response"`
	Count         json.Number       `json:"count"`
	Pagination    *publicPagination `json:"pagination"`
}
