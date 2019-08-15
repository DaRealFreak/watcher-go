package pixiv

import "encoding/json"

type user struct {
	ProfileImageUrls map[string]string `json:"profile_image_urls"`
	Id               string            `json:"id"`
	Name             string            `json:"name"`
	Account          string            `json:"account"`
	MailAddress      string            `json:"mail_address"`
	IsPremium        bool              `json:"is_premium"`
	XRestrict        json.Number       `json:"x_restrict"`
	IsMailAuthorized bool              `json:"is_mail_authorized"`
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

type loginResponse struct {
	Response *loginResponseData `json:"response"`
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
