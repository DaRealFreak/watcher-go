package fanboxapi

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/DaRealFreak/watcher-go/pkg/fp"
)

// FanboxUser contains the relevant information of the user passed from the API
type FanboxUser struct {
	UserID json.Number `json:"userId"`
	Name   string      `json:"name"`
}

// FanboxPost contains the relevant information on posts in the fanbox
type FanboxPost struct {
	ID    json.Number `json:"id"`
	Title string      `json:"title"`
	User  FanboxUser  `json:"user"`
}

// CreatorInfo contains all relevant information on the creator.
// The creator is no longer wrapped in an extra "creator" object but placed directly in the body.
// Error responses are handled by APIError/APIRequestError in mapAPIResponse.
type CreatorInfo struct {
	Body struct {
		User        FanboxUser `json:"user"`
		CreatorID   string     `json:"creatorId"`
		Description string     `json:"description"`
	} `json:"body"`
}

// PostPagination contains the list of paginated post list URLs of a creator
type PostPagination struct {
	Body struct {
		URLs []string `json:"pageUrls"`
	} `json:"body"`
}

// PostInfoSinglePage contains all relevant pixiv fanbox post info
type PostInfoSinglePage struct {
	Body struct {
		Posts []FanboxPost `json:"posts"`
	} `json:"body"`
}

// GetUserTag returns the default download tag for illustrations of the user context
func (u *FanboxUser) GetUserTag() string {
	return fmt.Sprintf("%s/%s", u.UserID.String(), fp.SanitizePath(u.Name, false))
}

// GetCreator requests the creator information from the unofficial fanbox/creator endpoint
func (a *FanboxAPI) GetCreator(creatorId string) (*CreatorInfo, error) {
	var info CreatorInfo

	res, err := a.get(fmt.Sprintf("https://api.fanbox.cc/creator.get?creatorId=%s", creatorId))
	if err != nil {
		return nil, err
	}

	if err = a.mapAPIResponse(res, &info); err != nil {
		return nil, err
	}

	return &info, nil
}

// GetPostPagination returns the post pagination list of the passed user
func (a *FanboxAPI) GetPostPagination(creatorId string) (*PostPagination, error) {
	values := url.Values{
		"creatorId": {creatorId},
	}

	apiURL := fmt.Sprintf("https://api.fanbox.cc/post.paginateCreator?%s", values.Encode())

	var postPagination PostPagination

	res, err := a.get(apiURL)
	if err != nil {
		return nil, err
	}

	if err = a.mapAPIResponse(res, &postPagination); err != nil {
		return nil, err
	}

	return &postPagination, nil
}

// GetPostList returns the post list of the passed user, starting at the passed cursor.
// The API requires firstPublishedDatetime and firstId to be passed together, so the cursor
// is only appended if both parts are set, otherwise the newest page is returned.
func (a *FanboxAPI) GetPostList(creatorId string, firstPublishedTime *time.Time, firstId int, limit int) (*PostInfoSinglePage, error) {
	values := url.Values{
		"creatorId": {creatorId},
		"sort":      {"newest"},
		"limit":     {strconv.Itoa(limit)},
	}

	if firstPublishedTime != nil && firstId > 0 {
		values.Add("firstPublishedDatetime", firstPublishedTime.Format("2006-01-02 15:04:05"))
		values.Add("firstId", strconv.Itoa(firstId))
	}

	apiURL := fmt.Sprintf("https://api.fanbox.cc/post.listCreator?%s", values.Encode())

	return a.GetPostListByURL(apiURL)
}

// GetPostListByURL returns the post info solely by the URL, used for the page URLs
// returned by the post pagination endpoint
func (a *FanboxAPI) GetPostListByURL(url string) (*PostInfoSinglePage, error) {
	var postInfoSinglePage PostInfoSinglePage

	res, err := a.get(url)
	if err != nil {
		return nil, err
	}

	if err = a.mapAPIResponse(res, &postInfoSinglePage); err != nil {
		return nil, err
	}

	return &postInfoSinglePage, nil
}
