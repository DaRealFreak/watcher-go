package api

import (
	"fmt"
	"net/url"
)

func (a *SankakuComplexApi) GetPoolResponse(postId string) (*ApiBookResponse, error) {
	apiURI := fmt.Sprintf(
		"https://sankakuapi.com/post/%s/pools?lang=en",
		url.QueryEscape(postId),
	)

	response, responseErr := a.get(apiURI)
	if responseErr != nil {
		return nil, responseErr
	}

	var booksResponse []ApiBookResponse
	if err := a.parseAPIResponse(response, &booksResponse); err != nil {
		return nil, err
	}

	if len(booksResponse) == 0 {
		return nil, nil
	}

	return &booksResponse[0], nil
}
