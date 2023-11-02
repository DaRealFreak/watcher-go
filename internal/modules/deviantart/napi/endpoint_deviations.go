package napi

import (
	"encoding/json"
	"github.com/DaRealFreak/watcher-go/internal/http"
	"net/url"
	"strconv"
)

type SearchResponse struct {
	HasMore        bool         `json:"hasMore"`
	NextCursor     string       `json:"nextCursor"`
	EstimatedTotal *json.Number `json:"estTotal"`
	CurrentOffset  *json.Number `json:"currentOffset"`
	Deviations     []*Deviation `json:"deviations"`
}

type ExtendedDeviationResponse struct {
	Deviation *Deviation `json:"deviation"`
	Session   struct {
		CSRFToken string `json:"csrfToken"`
	} `json:"session"`
}

const DeviationTypeArt = "art"
const DeviationTypeJournal = "journal"

func (a *DeviantartNAPI) ExtendedDeviation(
	deviationId int, username string, deviationType string, includeSession bool, session http.SessionInterface,
) (*ExtendedDeviationResponse, error) {
	if session == nil {
		session = a.UserSession
	}

	values := url.Values{
		"deviationid": {strconv.Itoa(deviationId)},
		"type":        {deviationType},
		"csrf_token":  {a.csrfToken},
	}

	if username != "" {
		values.Set("username", username)
	}

	if includeSession {
		values.Set("include_session", "true")
	} else {
		values.Set("include_session", "false")
	}

	apiUrl := "https://www.deviantart.com/_puppy/dadeviation/init?" + values.Encode()
	response, err := session.Get(apiUrl)
	if err != nil {
		return nil, err
	}

	var searchResponse ExtendedDeviationResponse
	err = a.mapAPIResponse(response, &searchResponse)

	// we most likely ran into an expired form submission which indicates that our session got terminated from the server
	if err == nil && searchResponse.Deviation == nil {
		// ToDo: investigate why the form submission is expired and how to refresh it
		// return a.ExtendedDeviation(deviationId, username, deviationType, includeSession, session)
	}

	return &searchResponse, err
}

func (a *DeviantartNAPI) DeviationSearch(search string, cursor string, order string) (*SearchResponse, error) {
	values := url.Values{
		"q": {search},
		// set order to most-recent by default, update if set later
		"order":      {OrderMostRecent},
		"csrf_token": {a.csrfToken},
	}

	if cursor != "" {
		values.Set("cursor", cursor)
	}

	if order != "" {
		values.Set("order", order)
	}

	apiUrl := "https://www.deviantart.com/_puppy/dabrowse/search/deviations?" + values.Encode()
	response, err := a.UserSession.Get(apiUrl)
	if err != nil {
		return nil, err
	}

	var searchResponse SearchResponse
	err = a.mapAPIResponse(response, &searchResponse)

	return &searchResponse, err
}

func (a *DeviantartNAPI) DeviationTag(tag string, cursor string, order string) (*SearchResponse, error) {
	values := url.Values{
		"tag": {tag},
		// set order to most-recent by default, update if set later
		"order":      {OrderMostRecent},
		"csrf_token": {a.csrfToken},
	}

	if cursor != "" {
		values.Set("cursor", cursor)
	}

	if order != "" {
		values.Set("order", order)
	}

	apiUrl := "https://www.deviantart.com/_puppy/dabrowse/networkbar/tag/deviations?" + values.Encode()
	response, err := a.UserSession.Get(apiUrl)
	if err != nil {
		return nil, err
	}

	var searchResponse SearchResponse
	err = a.mapAPIResponse(response, &searchResponse)

	return &searchResponse, err
}
