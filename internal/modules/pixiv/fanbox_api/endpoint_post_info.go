package fanboxapi

import (
	"encoding/json"
	"fmt"
)

// FanboxPostInfo contains the relevant fanbox post information
type FanboxPostInfo struct {
	Body struct {
		User struct {
			UserId string `json:"userId"`
			Name   string `json:"name"`
		} `json:"user"`
		PostBody struct {
			Text  string `json:"text"`
			Files []*struct {
				ID        string `json:"id"`
				Name      string `json:"name"`
				Extension string `json:"extension"`
				URL       string `json:"url"`
			}
			Images []*struct {
				ID           string `json:"id"`
				Extension    string `json:"extension"`
				OriginalURL  string `json:"originalUrl"`
				ThumbnailURL string `json:"thumbnailUrl"`
			} `json:"images"`
			Blocks []struct {
				Type    string `json:"type"`
				Text    string `json:"text"`
				ImageID string `json:"imageId"`
			} `json:"blocks"`
			ImageMap map[string]struct {
				ID          string `json:"id"`
				OriginalURL string `json:"originalUrl"`
			} `json:"imageMap"`
			FileMap map[string]struct {
				ID        string `json:"id"`
				Name      string `json:"name"`
				Extension string `json:"extension"`
				URL       string `json:"url"`
			} `json:"fileMap"`
		} `json:"body"`
		ID            json.Number `json:"id"`
		Title         string      `json:"title"`
		ImageForShare string      `json:"imageForShare"`
		CommentList   struct {
			Items []struct {
				ID   string `json:"id"`
				Body string `json:"body"`
				User struct {
					UserId string `json:"userId"`
					Name   string `json:"name"`
				} `json:"user"`
			} `json:"items"`
			NextUrl string `json:"nextUrl"`
		} `json:"commentList"`
	} `json:"body"`
}

func (i *FanboxPostInfo) CommentsFromAuthor() []string {
	var comments []string

	for _, comment := range i.Body.CommentList.Items {
		if comment.User.UserId == i.Body.User.UserId {
			comments = append(comments, comment.Body)
		}
	}

	return comments
}

// ImagesFromBlocks returns all image URLs from the Blocks section of the fanbox post
func (i *FanboxPostInfo) ImagesFromBlocks() []string {
	var imageURLs []string

	for _, block := range i.Body.PostBody.Blocks {
		if block.Type == "image" {
			imageURLs = append(imageURLs, i.Body.PostBody.ImageMap[block.ImageID].OriginalURL)
		}
	}

	return imageURLs
}

// GetPostInfo requests the fanbox post info from the API for the passed post ID
func (a *FanboxAPI) GetPostInfo(postID int) (*FanboxPostInfo, error) {
	var postInfo FanboxPostInfo

	res, err := a.get(fmt.Sprintf("https://api.fanbox.cc/post.info?postId=%d", postID))
	if err != nil {
		return nil, err
	}

	if err = a.mapAPIResponse(res, &postInfo); err != nil {
		return nil, err
	}

	return &postInfo, nil
}
