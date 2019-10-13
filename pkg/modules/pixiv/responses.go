package pixiv

import (
	"encoding/json"
	"time"
)

// user is the JSON struct of user objects returned by the API
type user struct {
	ProfileImageUrls map[string]string `json:"profile_image_urls"`
	ID               json.Number       `json:"id"`
	Name             string            `json:"name"`
	Account          string            `json:"account"`
	MailAddress      string            `json:"mail_address"`
	IsPremium        bool              `json:"is_premium"`
	IsMailAuthorized bool              `json:"is_mail_authorized"`
	IsFollowed       bool              `json:"is_followed"`
	XRestrict        json.Number       `json:"x_restrict"`
	Comment          string            `json:"comment"`
}

// illustration is the JSON struct of illustration objects returned by the API
type illustration struct {
	ID             json.Number                    `json:"id"`
	Title          string                         `json:"title"`
	Type           string                         `json:"type"`
	ImageUrls      map[string]string              `json:"image_urls"`
	Caption        string                         `json:"caption"`
	Restrict       json.Number                    `json:"restrict"`
	User           *user                          `json:"user"`
	Tags           []*tag                         `json:"tags"`
	Tools          []string                       `json:"tools"`
	CreateDate     time.Time                      `json:"create_date"`
	PageCount      json.Number                    `json:"page_count"`
	Width          json.Number                    `json:"width"`
	Height         json.Number                    `json:"height"`
	SanityLevel    json.Number                    `json:"sanity_level"`
	XRestrict      json.Number                    `json:"x_restrict"`
	Series         *series                        `json:"series"`
	MetaSinglePage map[string]string              `json:"meta_single_page"`
	MetaPages      []map[string]map[string]string `json:"meta_pages"`
	TotalView      json.Number                    `json:"total_view"`
	TotalBookmarks json.Number                    `json:"total_bookmarks"`
	IsBookmarked   bool                           `json:"is_bookmarked"`
	Visible        bool                           `json:"visible"`
	IsMuted        bool                           `json:"is_muted"`
	TotalComments  json.Number                    `json:"total_comments"`
}

// series is the JSON struct of series objects returned by the API
type series struct {
	ID    json.Number `json:"id"`
	Title string      `json:"title"`
}

// ugoiraMetadata is the JSON struct of ugoira metadata objects returned by the API
type ugoiraMetadata struct {
	ZipUrls map[string]string `json:"zip_urls"`
	Frames  []*frame
}

// frame is the JSON struct of frame objects returned by the API
type frame struct {
	File  string      `json:"file"`
	Delay json.Number `json:"delay"`
}

// tag is the JSON struct of tag objects returned by the API
type tag struct {
	Name           string `json:"name"`
	TranslatedName string `json:"translated_name"`
}

// profile is the JSON struct of profile objects returned by the API
type profile struct {
	Webpage                    string      `json:"webpage"`
	Gender                     string      `json:"gender"`
	Birth                      string      `json:"birth"`
	BirthDay                   string      `json:"birth_day"`
	BirthYear                  json.Number `json:"birth_year"`
	Region                     string      `json:"region"`
	AddressID                  json.Number `json:"address_id"`
	CountryCode                string      `json:"country_code"`
	Job                        string      `json:"job"`
	JobID                      json.Number `json:"job_id"`
	TotalFollowUsers           json.Number `json:"total_follow_users"`
	TotalPixivUsers            json.Number `json:"total_mypixiv_users"`
	TotalIllusts               json.Number `json:"total_illusts"`
	TotalManga                 json.Number `json:"total_manga"`
	TotalNovels                json.Number `json:"total_novels"`
	TotalIllustBookmarksPublic json.Number `json:"total_illust_bookmarks_public"`
	TotalIllustSeries          json.Number `json:"total_illust_series"`
	TotalNovelSeries           json.Number `json:"total_novel_series"`
	BackgroundImageURL         string      `json:"background_image_url"`
	TwitterAccount             string      `json:"twitter_account"`
	TwitterURL                 string      `json:"twitter_url"`
	PawooURL                   string      `json:"pawoo_url"`
	IsPremium                  bool        `json:"is_premium"`
	IsUsingCustomProfileImage  bool        `json:"is_using_custom_profile_image"`
}

// profilePublicity is the JSON struct of profile publicity objects returned by the API
type profilePublicity struct {
	Gender    string `json:"gender"`
	Region    string `json:"region"`
	BirthDay  string `json:"birth_day"`
	BirthYear string `json:"birth_year"`
	Job       string `json:"job"`
	Pawoo     bool   `json:"pawoo"`
}

// workspace is the JSON struct of workspace objects returned by the API
type workspace struct {
	Pc                string `json:"pc"`
	Monitor           string `json:"monitor"`
	Tool              string `json:"tool"`
	Scanner           string `json:"scanner"`
	Tablet            string `json:"tablet"`
	Mouse             string `json:"mouse"`
	Printer           string `json:"printer"`
	Desktop           string `json:"desktop"`
	Music             string `json:"music"`
	Desk              string `json:"desk"`
	Chair             string `json:"chair"`
	Comment           string `json:"comment"`
	WorkspaceImageURL string `json:"workspace_image_url"`
}

// loginResponseData is the JSON struct of login response data objects returned by the API
type loginResponseData struct {
	AccessToken  string      `json:"access_token"`
	ExpiresIn    json.Number `json:"expires_in"`
	TokenType    string      `json:"token_type"`
	Scope        string      `json:"scope"`
	RefreshToken string      `json:"refresh_token"`
	User         user        `json:"user"`
	DeviceToken  string      `json:"device_token"`
}

// errorMessage is the JSON struct of error message objects returned by the API
type errorMessage struct {
	Message string      `json:"message"`
	Code    json.Number `json:"code"`
}

// errorData is the JSON struct of error data objects returned by the API
type errorData struct {
	System *errorMessage `json:"system"`
}

// errorResponse is the JSON struct of error response objects returned by the API
type errorResponse struct {
	HasError bool       `json:"has_error"`
	Errors   *errorData `json:"errors"`
}

// loginResponse is the JSON struct of login response objects returned by the API
type loginResponse struct {
	Response *loginResponseData `json:"response"`
}

// userDetailResponse is the JSON struct of user detail response objects returned by the API
type userDetailResponse struct {
	User             *user             `json:"user"`
	Profile          *profile          `json:"profile"`
	ProfilePublicity *profilePublicity `json:"profile_publicity"`
	Workspace        *workspace        `json:"workspace"`
}

// userWorkResponse is the JSON struct of user work response objects returned by the API
type userWorkResponse struct {
	Illustrations []*illustration `json:"illusts"`
	NextURL       string          `json:"next_url"`
}

// illustrationDetailResponse is the JSON struct of illust detail response objects returned by the API
type illustrationDetailResponse struct {
	Illustration *illustration `json:"illust"`
}

// ugoiraResponse is the JSON struct of ugoira response objects returned by the API
type ugoiraResponse struct {
	UgoiraMetadata *ugoiraMetadata `json:"ugoira_metadata"`
}
