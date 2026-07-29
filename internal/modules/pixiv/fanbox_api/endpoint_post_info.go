package fanboxapi

import (
	"encoding/json"
	"fmt"
)

// commentLimit is the highest limit the post.getComments endpoint accepts, the endpoint
// requires the limit to be passed explicitly
const commentLimit = 100

// FanboxPostDetail contains the relevant information of a single fanbox post.
// PostBody is null for posts the current session has no access to.
type FanboxPostDetail struct {
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
		} `json:"files"`
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
}

// FanboxPostInfo contains the relevant fanbox post information
type FanboxPostInfo struct {
	Body struct {
		Post FanboxPostDetail `json:"post"`
	} `json:"body"`
}

// FanboxComment contains the relevant information of a single comment of a fanbox post,
// replies are nested comments of the same shape
type FanboxComment struct {
	ID   string `json:"id"`
	Body string `json:"body"`
	User struct {
		UserId string `json:"userId"`
		Name   string `json:"name"`
	} `json:"user"`
	Replies []FanboxComment `json:"replies"`
}

// FanboxPostComments contains the comment list of a fanbox post, which is no longer part
// of the post.info response but served by the separate post.getComments endpoint.
// CommentList is null if the current session may not view the comments.
type FanboxPostComments struct {
	Body struct {
		ViewMode    string `json:"viewMode"`
		CommentList *struct {
			Items   []FanboxComment `json:"items"`
			NextURL string          `json:"nextUrl"`
		} `json:"commentList"`
	} `json:"body"`
}

// CommentsFromAuthor returns the comment bodies written by the passed author, replies are
// included since authors commonly answer within the comment threads
func (c *FanboxPostComments) CommentsFromAuthor(authorUserId string) []string {
	if c.Body.CommentList == nil {
		return nil
	}

	return commentsFromAuthor(c.Body.CommentList.Items, authorUserId)
}

// commentsFromAuthor collects the comment bodies of the passed author from the comments
// and their replies
func commentsFromAuthor(comments []FanboxComment, authorUserId string) []string {
	var authorComments []string

	for _, comment := range comments {
		if comment.User.UserId == authorUserId {
			authorComments = append(authorComments, comment.Body)
		}

		authorComments = append(authorComments, commentsFromAuthor(comment.Replies, authorUserId)...)
	}

	return authorComments
}

// ImagesFromBlocks returns all image URLs from the Blocks section of the fanbox post
func (p *FanboxPostDetail) ImagesFromBlocks() []string {
	var imageURLs []string

	for _, block := range p.PostBody.Blocks {
		if block.Type == "image" {
			imageURLs = append(imageURLs, p.PostBody.ImageMap[block.ImageID].OriginalURL)
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

// GetPostComments requests the comments of the passed post ID from the API
func (a *FanboxAPI) GetPostComments(postID int) (*FanboxPostComments, error) {
	var comments FanboxPostComments

	res, err := a.get(fmt.Sprintf("https://api.fanbox.cc/post.getComments?postId=%d&limit=%d", postID, commentLimit))
	if err != nil {
		return nil, err
	}

	if err = a.mapAPIResponse(res, &comments); err != nil {
		return nil, err
	}

	return &comments, nil
}
