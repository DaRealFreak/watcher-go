package napi

import (
	"encoding/json"
	"net/url"
	"strconv"
)

type Author struct {
	UserId     json.Number `json:"userId"`
	UseridUuid string      `json:"useridUuid"`
	Username   string      `json:"username"`
}

type Deviation struct {
	DeviationId    json.Number `json:"deviationId"`
	Type           string      `json:"type"`
	TypeID         json.Number `json:"typeId"`
	URL            string      `json:"url"`
	Title          string      `json:"title"`
	IsJournal      bool        `json:"isJournal"`
	IsVideo        bool        `json:"isVideo"`
	PublishedTime  string      `json:"publishedTime"`
	IsDeleted      bool        `json:"isDeleted"`
	IsDownloadable bool        `json:"isDownloadable"`
	IsBlocked      bool        `json:"isBlocked"`
	Author         *Author     `json:"author"`
	Media          *Media      `json:"media"`
}

type Media struct {
	BaseUri    string       `json:"baseUri"`
	PrettyName string       `json:"prettyName"`
	Types      []*MediaType `json:"types"`
}

type MediaType struct {
	Types    string       `json:"t"`
	Height   json.Number  `json:"h"`
	Width    json.Number  `json:"w"`
	Quality  *json.Number `json:"q"`
	FileSize *json.Number `json:"f"`
	URL      *string      `json:"b"`
}

type SearchResponse struct {
	HasMore        bool        `json:"hasMore"`
	NextCursor     string      `json:"nextCursor"`
	EstimatedTotal json.Number `json:"estTotal"`
	CurrentOffset  json.Number `json:"currentOffset"`
	Deviations     []struct {
	} `json:"deviations"`
}

func (m *Media) GetHighestQualityVideoType() (bestMediaType *MediaType) {
	fileSize := 0
	for _, mediaType := range m.Types {
		if mediaType.Types != "video" {
			continue
		}

		typeFileSize, _ := strconv.ParseInt(mediaType.FileSize.String(), 10, 64)
		if int(typeFileSize) >= fileSize {
			bestMediaType = mediaType
		}
	}

	return bestMediaType
}

func (a *DeviantartNAPI) DeviationSearch(search string, cursor string, order string) (*SearchResponse, error) {
	values := url.Values{
		"q": {search},
		// set order to most-recent by default, update if set later
		"order": {"most-recent"},
	}

	if cursor != "" {
		values.Set("cursor", cursor)
	}

	if order != "" {
		values.Set("order", order)
	}

	apiUrl := "https://www.deviantart.com/_napi/da-browse/api/networkbar/search/deviations?" + values.Encode()
	response, err := a.Session.Get(apiUrl)
	if err != nil {
		return nil, err
	}

	var searchResponse SearchResponse
	err = a.mapAPIResponse(response, &searchResponse)

	return &searchResponse, err

}
