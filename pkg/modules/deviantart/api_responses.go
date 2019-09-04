package deviantart

import (
	"encoding/json"
)

// loginInfo contains every JSON encoded information on the login page
type loginInfo struct {
	// ToDo: @@publicSession
	// ToDo: cardDeviation
	// ToDo: flags
	// ToDo: forgot
	// ToDo: join
	// ToDo: login (contains captcha requirement!)
	// ToDo: params
	// ToDo: perimeterx
	AuthMode            string `json:"authMode"`
	BaseDaURL           string `json:"baseDaUrl"`
	CardImage           string `json:"cardImage"`
	CSRFToken           string `json:"csrfToken"`
	EnvironmentType     string `json:"environmentType"`
	FacebookAppID       string `json:"facebookAppId"`
	GoogleClientID      string `json:"googleClientId"`
	RecaptchaSiteKey    string `json:"recaptchaSiteKey"`
	Referer             string `json:"referer"`
	RequestID           string `json:"requestId"`
	RootPath            string `json:"rootPath"`
	ShowCaptcha         bool   `json:"showCaptcha"`
	IsDebug             bool   `json:"isDebug"`
	IsMobile            bool   `json:"isMobile"`
	IsOauthError        bool   `json:"isOauthError"`
	IsTestEnv           bool   `json:"isTestEnv"`
	SocialBlockExpanded bool   `json:"socialBlockExpanded"`
}

// UtilPlaceboResponse is the struct for API endpoint https://www.deviantart.com/api/v1/oauth2/placebo
type UtilPlaceboResponse struct {
	Status string `json:"status"`
}

// CollectionsResponse is the struct for API endpoint https://www.deviantart.com/api/v1/oauth2/collections/{folderid}
type CollectionsResponse struct {
	DeviationPagination
}

// GalleryResponse is the struct for API endpoint https://www.deviantart.com/api/v1/oauth2/gallery/{folderid}
type GalleryResponse struct {
	DeviationPagination
}

// GalleryAllResponse is the struct for API endpoint https://www.deviantart.com/api/v1/oauth2/gallery/all
type GalleryAllResponse struct {
	DeviationPagination
}

// BrowseCategoryTreeResponse is the struct
// for API endpoint https://www.deviantart.com/api/v1/oauth2/browse/categorytree
type BrowseCategoryTreeResponse struct {
	Categories []*Category `json:"categories"`
}

// BrowseDailyDeviationsResponse is the struct
// for API endpoint https://www.deviantart.com/api/v1/oauth2/browse/dailydeviations
type BrowseDailyDeviationsResponse struct {
	Results []*Deviation `json:"results"`
}

// BrowseHotResponse is the struct
// for API endpoint https://www.deviantart.com/api/v1/oauth2/browse/hot
type BrowseHotResponse struct {
	DeviationPagination
}

// BrowseMoreLikeThisResponse is the struct
// for API endpoint https://www.deviantart.com/api/v1/oauth2/browse/morelikethis
type BrowseMoreLikeThisResponse struct {
	DeviationPagination
}

// BrowseMoreLikeThisPreviewResponse is the struct
// for API endpoint https://www.deviantart.com/api/v1/oauth2/browse/morelikethis/preview
type BrowseMoreLikeThisPreviewResponse struct {
	Seed           json.Number  `json:"seed"`
	Author         *User        `json:"author"`
	MoreFromArtist []*Deviation `json:"more_from_artist"`
	MoreFromDa     []*Deviation `json:"more_from_da"`
}

// BrowseNewestResponse is the struct for API endpoint https://www.deviantart.com/api/v1/oauth2/browse/newest
type BrowseNewestResponse struct {
	DeviationPagination
}

// BrowsePopularResponse is the struct for API endpoint https://www.deviantart.com/api/v1/oauth2/browse/popular
type BrowsePopularResponse struct {
	DeviationPagination
}

// BrowseTagsResponse is the struct for API endpoint https://www.deviantart.com/api/v1/oauth2/browse/tags
type BrowseTagsResponse struct {
	DeviationPagination
}

// BrowseTagsSearchResponse is the struct for API endpoint https://www.deviantart.com/api/v1/oauth2/browse/tags/search
type BrowseTagsSearchResponse struct {
	Results []*Tag `json:"results"`
}

// BrowseUndiscoveredResponse is the struct
// for API endpoint https://www.deviantart.com/api/v1/oauth2/browse/undiscovered
type BrowseUndiscoveredResponse struct {
	DeviationPagination
}

// BrowseUserJournalsResponse is the struct
// for API endpoint https://www.deviantart.com/api/v1/oauth2/browse/user/journals
type BrowseUserJournalsResponse struct {
	DeviationPagination
}

// DeviationContent is the struct for API endpoint https://www.deviantart.com/api/v1/oauth2/deviation/content
type DeviationContent struct {
	HTML     string `json:"html"`
	CSS      string `json:"css"`
	CSSFonts string `json:"css_fonts"`
}

// GalleryFoldersCreateResponse is the struct
// for API endpoint https://www.deviantart.com/api/v1/oauth2/gallery/folders/create
type GalleryFoldersCreateResponse struct {
	FolderID json.Number `json:"folderid"`
	Name     string      `json:"name"`
}

// FeedBucketResponse is the struct
// for API endpoint https://www.deviantart.com/api/v1/oauth2/feed/home/{bucketid}
type FeedBucketResponse struct {
	Cursor  string      `json:"cursor"`
	HasMore bool        `json:"has_more"`
	Items   []*FeedItem `json:"items"`
}
