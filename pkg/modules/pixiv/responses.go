package pixiv

import (
	"encoding/json"
	"time"
)

type user struct {
	ProfileImageUrls map[string]string `json:"profile_image_urls"`
	Id               json.Number       `json:"id"`
	Name             string            `json:"name"`
	Account          string            `json:"account"`
	MailAddress      string            `json:"mail_address"`
	IsPremium        bool              `json:"is_premium"`
	XRestrict        json.Number       `json:"x_restrict"`
	IsMailAuthorized bool              `json:"is_mail_authorized"`
	Comment          string            `json:"comment"`
	IsFollowed       bool              `json:"is_followed"`
}

type illustration struct {
	Id             json.Number                    `json:"id"`
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

type series struct {
	Id    json.Number `json:"id"`
	Title string      `json:"title"`
}

type ugoiraMetadata struct {
	ZipUrls map[string]string `json:"zip_urls"`
	Frames  []*frame
}

type frame struct {
	File  string      `json:"file"`
	Delay json.Number `json:"delay"`
}

type tag struct {
	Name           string `json:"name"`
	TranslatedName string `json:"translated_name"`
}

type profile struct {
	Webpage                    string      `json:"webpage"`
	Gender                     string      `json:"gender"`
	Birth                      string      `json:"birth"`
	BirthDay                   string      `json:"birth_day"`
	BirthYear                  json.Number `json:"birth_year"`
	Region                     string      `json:"region"`
	AddressId                  json.Number `json:"address_id"`
	CountryCode                string      `json:"country_code"`
	Job                        string      `json:"job"`
	JobId                      json.Number `json:"job_id"`
	TotalFollowUsers           json.Number `json:"total_follow_users"`
	TotalPixivUsers            json.Number `json:"total_mypixiv_users"`
	TotalIllusts               json.Number `json:"total_illusts"`
	TotalManga                 json.Number `json:"total_manga"`
	TotalNovels                json.Number `json:"total_novels"`
	TotalIllustBookmarksPublic json.Number `json:"total_illust_bookmarks_public"`
	TotalIllustSeries          json.Number `json:"total_illust_series"`
	TotalNovelSeries           json.Number `json:"total_novel_series"`
	BackgroundImageUrl         string      `json:"background_image_url"`
	TwitterAccount             string      `json:"twitter_account"`
	TwitterUrl                 string      `json:"twitter_url"`
	PawooUrl                   string      `json:"pawoo_url"`
	IsPremium                  bool        `json:"is_premium"`
	IsUsingCustomProfileImage  bool        `json:"is_using_custom_profile_image"`
}

type profilePublicity struct {
	Gender    string `json:"gender"`
	Region    string `json:"region"`
	BirthDay  string `json:"birth_day"`
	BirthYear string `json:"birth_year"`
	Job       string `json:"job"`
	Pawoo     bool   `json:"pawoo"`
}

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
	WorkspaceImageUrl string `json:"workspace_image_url"`
}

type loginResponseData struct {
	AccessToken  string      `json:"access_token"`
	ExpiresIn    json.Number `json:"expires_in"`
	TokenType    string      `json:"token_type"`
	Scope        string      `json:"scope"`
	RefreshToken string      `json:"refresh_token"`
	User         user        `json:"user"`
	DeviceToken  string      `json:"device_token"`
}

type errorMessage struct {
	Message string      `json:"message"`
	Code    json.Number `json:"code"`
}

type errorData struct {
	System *errorMessage `json:"system"`
}

type errorResponse struct {
	HasError bool       `json:"has_error"`
	Errors   *errorData `json:"errors"`
}

type loginResponse struct {
	Response *loginResponseData `json:"response"`
}

type userDetailResponse struct {
	User             *user             `json:"user"`
	Profile          *profile          `json:"profile"`
	ProfilePublicity *profilePublicity `json:"profile_publicity"`
	Workspace        *workspace        `json:"workspace"`
}

type userWorkResponse struct {
	Illustrations []*illustration `json:"illusts"`
	NextUrl       string          `json:"next_url"`
}

type ugoiraResponse struct {
	UgoiraMetadata *ugoiraMetadata `json:"ugoira_metadata"`
}
