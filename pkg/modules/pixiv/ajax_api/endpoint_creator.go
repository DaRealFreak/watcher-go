package ajaxapi

import (
	"encoding/json"
	"fmt"
)

// FanboxUser contains the relevant information of the user passed from the API
type FanboxUser struct {
	UserID string `json:"userId"`
	Name   string `json:"name"`
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
		Post struct {
			Items   []FanboxPost `json:"items"`
			NextURL string       `json:"nextUrl"`
		} `json:"post"`
	} `json:"body"`
	Error   bool   `json:"error"`
	Message string `json:"message"`
}

// PostInfo contains all relevant pixiv fanbox post info
type PostInfo struct {
	Body struct {
		Post struct {
			Items   []FanboxPost `json:"items"`
			NextURL string       `json:"nextUrl"`
		} `json:"post"`
	} `json:"body"`
}

// GetCreator requests the creator information from the unofficial ajax/fanbox/creator endpoint
func (a *AjaxAPI) GetCreator(userID int) (*CreatorInfo, error) {
	var info CreatorInfo

	res, err := a.Session.Get(fmt.Sprintf("https://www.pixiv.net/ajax/fanbox/creator?userId=%d", userID))
	if err != nil {
		return nil, err
	}

	if err := a.mapAPIResponse(res, &info); err != nil {
		return nil, err
	}

	return &info, nil
}

// GetPostList returns the initial post list of the passed user
func (a *AjaxAPI) GetPostList(userID int) (*PostInfo, error) {
	var postInfo PostInfo

	url := fmt.Sprintf("https://fanbox.pixiv.net/api/post.listCreator?userId=%d", userID)

	res, err := a.Session.Get(url)
	if err != nil {
		panic(err)
	}

	if err := a.mapAPIResponse(res, &postInfo); err != nil {
		return nil, err
	}

	return &postInfo, nil
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
