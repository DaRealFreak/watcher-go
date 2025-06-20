package api

import (
	"fmt"
	"net/url"
)

type BookApiResponse struct {
	Pools  poolsResponse  `json:"pools"`
	Series seriesResponse `json:"series"`
	Page   int            `json:"page"`
}

type poolsResponse struct {
	Meta apiMeta        `json:"meta"`
	Data []*BookApiItem `json:"data"`
}

// BookApiItem is the JSON struct of item objects returned by the API
type BookApiItem struct {
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
	Tags             []*ApiTag  `json:"tags"`
	PostTags         []*ApiTag  `json:"post_tags"`
	ArtistTags       []*ApiTag  `json:"artist_tags"`
	GenreTags        []*ApiTag  `json:"genre_tags"`
	Posts            []*ApiItem `json:"posts"`
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

func (b BookApiItem) GetPostIndex(id string) int {
	for i, post := range b.Posts {
		if post.ID == id {
			return i
		}
	}

	return -1
}

type seriesResponse struct {
	Data       []series `json:"data"`
	TotalCount int      `json:"totalCount"`
}

type series struct {
}

func (a *SankakuComplexApi) GetFilledBookResponse(tag string, page int) (*BookApiResponse, error) {
	apiURI := fmt.Sprintf(
		"https://sankakuapi.com/poolseriesv2?lang=en&filledPools=true&offset=0&limit=100&tags=order:date+%s&page=%d&includes[]=pools&exceptStatuses[]=deleted",
		url.QueryEscape(tag),
		page,
	)

	response, err := a.get(apiURI)
	if err != nil {
		return nil, err
	}

	var apiGalleryResponse BookApiResponse
	if err = a.parseAPIResponse(response, &apiGalleryResponse); err != nil {
		return nil, err
	}

	return &apiGalleryResponse, nil
}

func (a *SankakuComplexApi) GetBookResponse(bookId string) (*BookApiItem, error) {
	apiURI := fmt.Sprintf(
		"https://sankakuapi.com/pools/%s?lang=en&includes[]=series",
		url.QueryEscape(bookId),
	)

	response, responseErr := a.get(apiURI)
	if responseErr != nil {
		return nil, responseErr
	}

	var booksResponse BookApiItem
	if err := a.parseAPIResponse(response, &booksResponse); err != nil {
		return nil, err
	}

	return &booksResponse, nil
}
