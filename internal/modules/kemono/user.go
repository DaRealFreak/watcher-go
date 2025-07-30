package kemono

import (
	"fmt"
	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules/kemono/api"
	"regexp"
)

func (m *kemono) parseUser(item *models.TrackedItem) error {
	search := regexp.MustCompile(`https://(?:kemono|coomer).\w+/([^/?&]+)/user/([^/?&]+)`).FindStringSubmatch(item.URI)
	userId := ""
	service := ""
	if len(search) == 3 {
		service = search[1]
		userId = search[2]
	}

	if userId == "" || service == "" {
		return fmt.Errorf("could not extract user ID and service from URL: %s", item.URI)
	}

	root, err := m.api.GetUserPosts(service, userId, 0)
	if err != nil {
		return fmt.Errorf("failed to fetch user posts: %w", err)
	}

	var (
		downloadQueue    []api.Result
		foundCurrentItem bool
		offset           int
	)

	for len(root.Results) != 0 {
		for _, post := range root.Results {
			// check if we reached the current item already
			if post.ID == item.CurrentItem {
				foundCurrentItem = true
				break
			}

			downloadQueue = append(downloadQueue, post)
		}

		if foundCurrentItem {
			break
		}

		// increase offset for the next page
		offset += 50
		maxCount, _ := root.Properties.Count.Int64()
		if offset >= int(maxCount) {
			break
		}

		root, err = m.api.GetUserPosts(service, userId, offset)
		if err != nil {
			return fmt.Errorf("failed to unmarshal JSON: %w", err)
		}
	}

	// reverse download queue to download old items first
	for i, j := 0, len(downloadQueue)-1; i < j; i, j = i+1, j-1 {
		downloadQueue[i], downloadQueue[j] = downloadQueue[j], downloadQueue[i]
	}

	return m.processDownloadQueue(item, downloadQueue)
}

func (m *kemono) parsePost(item *models.TrackedItem) error {
	// extract 2nd number from example URL: https://kemono.su/patreon/user/551274/post/24446001
	postId := regexp.MustCompile(`.*/([^/?&]+)/user/([^/?&]+)/post/(\w+)`).FindStringSubmatch(item.URI)
	if len(postId) != 4 {
		return fmt.Errorf("could not extract post ID from URL: %s", item.URI)
	}

	return m.processDownloadQueue(item, []api.Result{{
		Service: postId[1],
		User:    postId[2],
		ID:      postId[3],
	}})
}
