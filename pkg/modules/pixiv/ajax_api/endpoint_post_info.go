package ajaxapi

import (
	"encoding/json"
	"fmt"
)

// FanboxPostInfo contains the relevant fanbox post information
type FanboxPostInfo struct {
	Body struct {
		PostBody struct {
			Text   string `json:"text"`
			Images []*struct {
				ID           string `json:"id"`
				Extension    string `json:"extension"`
				OriginalURL  string `json:"originalUrl"`
				ThumbnailURL string `json:"thumbnailUrl"`
			} `json:"images"`
		} `json:"body"`
		ID            json.Number `json:"id"`
		Title         string      `json:"title"`
		ImageForShare string      `json:"imageForShare"`
	} `json:"body"`
}

// GetPostInfo requests the fanbox post info from the API for the passed post ID
func (a *AjaxAPI) GetPostInfo(postID int) (*FanboxPostInfo, error) {
	var postInfo FanboxPostInfo

	res, err := a.Session.Get(fmt.Sprintf("https://fanbox.pixiv.net/api/post.info?postId=%d", postID))
	if err != nil {
		return nil, err
	}

	if err := a.mapAPIResponse(res, &postInfo); err != nil {
		return nil, err
	}

	return &postInfo, nil
}
