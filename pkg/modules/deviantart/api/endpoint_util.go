package api

import "net/url"

// Placebo contains all relevant information of the API response of the placebo function
type Placebo struct {
	Status string `json:"status"`
}

// Placebo implements the API endpoint https://www.deviantart.com/api/v1/oauth2/placebo
func (a *DeviantartAPI) Placebo() (*Placebo, error) {
	a.ApplyRateLimit()

	res, err := a.request("GET", "/placebo", url.Values{})
	if err != nil {
		return nil, err
	}

	var placebo Placebo
	err = a.mapAPIResponse(res, &placebo)

	return &placebo, err
}
