package api

import "fmt"

type Tag struct {
	Tag struct {
		Total       int `json:"total"`
		PerPage     int `json:"per_page"`
		Pages       int `json:"pages"`
		CurrentPage int `json:"current_page"`
		Items       struct {
			ID     int    `json:"id"`
			Tag    string `json:"tag"`
			Images []struct {
				ID       int    `json:"id"`
				Filename string `json:"filename"`
			} `json:"images"`
		} `json:"items"`
	} `json:"tag"`
}

// RetrieveTagResponse retrieves and parses the request from the tag API URL
func (a *ChounyuuAPI) RetrieveTagResponse(domain string, tagId string, page int) (*Tag, error) {
	url := fmt.Sprintf(
		"https://g.%s/api/get/tag/%d/%s/%d",
		domain, a.getApiVersion(domain), tagId, page,
	)

	res, err := a.Session.Get(url)
	if err != nil {
		return nil, err
	}

	var tag Tag
	err = a.mapAPIResponse(res, &tag)

	return &tag, err
}
