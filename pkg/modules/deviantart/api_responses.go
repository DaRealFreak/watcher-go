package deviantart

import "encoding/json"

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

// GalleryAllResponse is the struct for API endpoint https://www.deviantart.com/api/v1/oauth2/gallery/all
type GalleryAllResponse struct {
	HasMore    bool        `json:"has_more"`
	NextOffset json.Number `json:"next_offset"`
	Results    []*Deviation
}

// GalleryAllResponse is the struct for API endpoint https://www.deviantart.com/api/v1/oauth2/browse/categorytree
type BrowseCategoryTreeResponse struct {
	Categories []*Category `json:"categories"`
}

// GalleryAllResponse is the struct
// for API endpoint https://www.deviantart.com/api/v1/oauth2/gallery/folders/create
type GalleryFoldersCreateResponse struct {
	FolderID json.Number `json:"folderid"`
	Name     string      `json:"name"`
}
