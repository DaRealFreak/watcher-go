package deviantart

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

// BrowseTagsResponse is the struct for API endpoint https://www.deviantart.com/api/v1/oauth2/browse/tags
type BrowseTagsResponse struct {
	DeviationPagination
}

// DeviationContent is the struct for API endpoint https://www.deviantart.com/api/v1/oauth2/deviation/content
type DeviationContent struct {
	HTML     string `json:"html"`
	CSS      string `json:"css"`
	CSSFonts string `json:"css_fonts"`
}

// FeedBucketResponse is the struct
// for API endpoint https://www.deviantart.com/api/v1/oauth2/feed/home/{bucketid}
type FeedBucketResponse struct {
	Cursor  string      `json:"cursor"`
	HasMore bool        `json:"has_more"`
	Items   []*FeedItem `json:"items"`
}
