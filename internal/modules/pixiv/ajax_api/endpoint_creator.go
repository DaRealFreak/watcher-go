package ajaxapi

import (
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"time"

	"github.com/DaRealFreak/watcher-go/internal/models"
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
	Type  string      `json:"type"`
	User  FanboxUser  `json:"user"`
}

// CreatorInfo contains all relevant information on the creator
type CreatorInfo struct {
	Body struct {
		Creator struct {
			User        FanboxUser `json:"user"`
			Description string     `json:"description"`
		} `json:"creator"`
	} `json:"body"`
	Error   bool   `json:"error"`
	Message string `json:"message"`
}

// PostInfo contains all relevant pixiv fanbox post info
type PostInfo struct {
	Body struct {
		Items   []FanboxPost `json:"items"`
		NextURL string       `json:"nextUrl"`
	} `json:"body"`
}

// GetUserTag returns the default download tag for illustrations of the user context
func (u *FanboxUser) GetUserTag() string {
	return fmt.Sprintf("%s/%s", u.UserID.String(), models.Module{}.SanitizePath(u.Name, false))
}

// GetCreator requests the creator information from the unofficial ajax/fanbox/creator endpoint
func (a *AjaxAPI) GetCreator(creatorId string) (*CreatorInfo, error) {
	var info CreatorInfo

	res, err := a.Session.Get(fmt.Sprintf("https://api.fanbox.cc/creator.get?creatorId=%s", creatorId))
	if err != nil {
		return nil, err
	}

	if err := a.mapAPIResponse(res, &info); err != nil {
		return nil, err
	}

	return &info, nil
}

// GetPostList returns the initial post list of the passed user
func (a *AjaxAPI) GetPostList(creatorId string, limit int) (*PostInfo, error) {
	values := url.Values{
		"creatorId":            {creatorId},
		"maxPublishedDatetime": {time.Now().Format("2006-01-02 15:04:05")},
		"maxId":                {strconv.Itoa(math.MaxUint32)},
		"limit":                {strconv.Itoa(limit)},
	}
	apiURL := fmt.Sprintf("https://api.fanbox.cc/post.listCreator?%s", values.Encode())

	return a.GetPostListByURL(apiURL)
}

// GetPostListByURL returns the post info solely by the URL since the PostInfo objects contain a NextURL string
func (a *AjaxAPI) GetPostListByURL(url string) (*PostInfo, error) {
	var postInfo PostInfo

	res, err := a.Session.Get(url)
	if err != nil {
		panic(err)
	}

	if err := a.mapAPIResponse(res, &postInfo); err != nil {
		return nil, err
	}

	return &postInfo, nil
}
