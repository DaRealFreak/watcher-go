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
	CsrfToken           string `json:"csrfToken"`
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

// authInfo contains all required information for the app authorization request
type authInfo struct {
	State         string
	Scope         string
	ResponseType  string
	ValidateToken string
	ValidateKey   string
}
