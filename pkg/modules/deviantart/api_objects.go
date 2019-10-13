package deviantart

import (
	"encoding/json"
)

// APIError is the struct of the error API responses
type APIError struct {
	Error            string            `json:"error"`
	ErrorDescription string            `json:"error_description"`
	ErrorDetails     map[string]string `json:"error_details"`
	ErrorCode        json.Number       `json:"error_code"`
	Status           string            `json:"status"`
}

// User is the struct of the user API response
// https://www.deviantart.com/developers/http/v1/20160316/object/user
type User struct {
	UserID     json.Number  `json:"userid"`
	Username   string       `json:"username"`
	UserIcon   string       `json:"usericon"`
	Type       string       `json:"type"`
	IsWatching bool         `json:"is_watching"`
	Details    *UserDetails `json:"details"`
	Geo        *GeoData     `json:"geo"`
	Profile    *UserProfile `json:"profile"`
	Stats      *UserStats   `json:"stats"`
}

// UserDetails is the struct of the user details API response
type UserDetails struct {
	Sex      string       `json:"sex"`
	Age      *json.Number `json:"age"`
	JoinDate string       `json:"joindate"`
}

// GeoData is the struct of the geo data API response
type GeoData struct {
	Country   string      `json:"country"`
	CountryID json.Number `json:"countryid"`
	Timezone  string      `json:"timezone"`
}

// UserProfile is the struct of the user profile API response
type UserProfile struct {
	UserIsArtist     bool       `json:"user_is_artist"`
	ArtistLevel      string     `json:"artist_level"`
	ArtistSpeciality string     `json:"artist_speciality"`
	RealName         string     `json:"real_name"`
	TagLine          string     `json:"tagline"`
	Website          string     `json:"website"`
	CoverPhoto       string     `json:"cover_photo"`
	ProfilePic       *Deviation `json:"profile_pic"`
}

// UserStats is the struct of the user stats API response
type UserStats struct {
	Watchers json.Number `json:"watchers"`
	Friends  json.Number `json:"friends"`
}

// DeviationStats is the struct of the deviation stats API response
type DeviationStats struct {
	Comments   json.Number `json:"comments"`
	Favourites json.Number `json:"favourites"`
}

// Image is the struct of the image API response
type Image struct {
	Src          string      `json:"src"`
	Height       json.Number `json:"height"`
	Width        json.Number `json:"width"`
	Transparency bool        `json:"transparency"`
	FileSize     json.Number `json:"filesize"`
}

// Video is the struct of the video API response
type Video struct {
	Src      string      `json:"src"`
	Quality  string      `json:"quality"`
	FileSize json.Number `json:"filesize"`
	Duration json.Number `json:"duration"`
}

// DailyDeviation is the struct of the daily deviation API response
type DailyDeviation struct {
	Body      string `json:"body"`
	Time      string `json:"time"`
	Giver     *User  `json:"giver"`
	Suggester *User  `json:"suggester"`
}

// Challenge is the struct of the challenge API response
type Challenge struct {
	Type            []string      `json:"type"`
	Tags            []string      `json:"tags"`
	CreditDeviation json.Number   `json:"credit_deviation"`
	Media           []string      `json:"media"`
	LevelLabel      string        `json:"level_label"`
	TimeLimit       json.Number   `json:"time_limit"`
	Levels          []json.Number `json:"levels"`
	Completed       bool          `json:"completed"`
	Locked          bool          `json:"locked"`
}

// ChallengeEntry is the struct of the challenge entry API response
type ChallengeEntry struct {
	ChallengeID    json.Number `json:"challengeid"`
	ChallengeTitle string      `json:"challenge_title"`
	Challenge      *Challenge  `json:"challenge"`
	TimedDuration  json.Number `json:"timed_duration"`
	SubmissionTime string      `json:"submission_time"`
}

// MotionBook is the struct of the motion book API response
type MotionBook struct {
	EmbedURL string `json:"embed_url"`
}

// Deviation is the struct of the deviation API response
// https://www.deviantart.com/developers/http/v1/20160316/object/deviation
type Deviation struct {
	DeviationID      json.Number     `json:"deviationid"`
	PrintID          json.Number     `json:"printid"`
	URL              string          `json:"url"`
	Title            string          `json:"title"`
	Category         string          `json:"category"`
	CategoryPath     string          `json:"category_path"`
	Author           *User           `json:"author"`
	Stats            *DeviationStats `json:"stats"`
	PublishedTime    string          `json:"published_time"`
	Preview          *Image          `json:"preview"`
	Content          *Image          `json:"content"`
	Thumbs           []*Image        `json:"thumbs"`
	Videos           []*Video        `json:"videos"`
	Flash            *Image          `json:"flash"`
	DailyDeviation   *DailyDeviation `json:"daily_deviation"`
	Excerpt          string          `json:"excerpt"`
	DownloadFileSize json.Number     `json:"download_filesize"`
	Challenge        *Challenge      `json:"challenge"`
	ChallengeEntry   *ChallengeEntry `json:"challenge_entry"`
	MotionBook       *MotionBook     `json:"motion_book"`
	IsFavourited     bool            `json:"is_favourited"`
	IsDeleted        bool            `json:"is_deleted"`
	AllowsComments   bool            `json:"allows_comments"`
	IsMature         bool            `json:"is_mature"`
	IsDownloadable   bool            `json:"is_downloadable"`
}

// DeviationPagination is the struct of deviation pagination API responses
type DeviationPagination struct {
	HasMore        bool         `json:"has_more"`
	NextOffset     json.Number  `json:"next_offset"`
	Results        []*Deviation `json:"results"`
	EstimatedTotal json.Number  `json:"estimated_total"`
}

// FeedItem is the struct of feed items API responses
type FeedItem struct {
	Timestamp  string       `json:"ts"`
	Type       string       `json:"type"`
	ByUser     *User        `json:"by_user"`
	Deviations []*Deviation `json:"deviations"`
}
