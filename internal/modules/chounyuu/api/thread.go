package api

import "fmt"

type Thread struct {
	Thread struct {
		Total       int `json:"total"`
		PerPage     int `json:"per_page"`
		Pages       int `json:"pages"`
		CurrentPage int `json:"current_page"`
		Items       struct {
			ID     int    `json:"id"`
			Title  string `json:"title"`
			Closed bool   `json:"closed"`
			Posts  []struct {
				ID       int    `json:"id"`
				Filename string `json:"filename"`
				ImageID  int    `json:"img_id"`
			} `json:"posts"`
		} `json:"items"`
	} `json:"thread"`
}

// RetrieveThreadResponse retrieves and parses the request from the thread API URL
func (a *ChounyuuAPI) RetrieveThreadResponse(domain string, threadId string, page int) (*Thread, error) {
	url := fmt.Sprintf(
		"https://g.%s/api/get/thread/%d/%s/%d",
		domain, a.getApiVersion(domain), threadId, page,
	)

	res, err := a.Session.Get(url)
	if err != nil {
		return nil, err
	}

	var thread Thread
	err = a.mapAPIResponse(res, &thread)

	return &thread, err
}
