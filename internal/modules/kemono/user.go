package kemono

import (
	"fmt"
	"regexp"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules/kemono/api"
	log "github.com/sirupsen/logrus"
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

	profile, err := m.api.GetUserProfile(service, userId)
	if err != nil {
		return fmt.Errorf("failed to fetch user profile: %w", err)
	}

	log.WithField("module", m.Key).Info(
		fmt.Sprintf("Found user %s (%s) with %d posts", profile.Name, profile.ID, profile.PostCount),
	)

	userPosts, err := m.api.GetUserPosts(service, userId, 0)
	if err != nil {
		return fmt.Errorf("failed to fetch user posts: %w", err)
	}

	var (
		downloadQueue    []api.QuickPost
		foundCurrentItem bool
		offset           int
	)

	for {
		for _, post := range userPosts {
			// check if we reached the current item already
			if post.ID == item.CurrentItem {
				foundCurrentItem = true
				break
			}

			downloadQueue = append(downloadQueue, post)
		}

		if foundCurrentItem || profile.PostCount <= offset+50 {
			break
		}

		// increase offset for the next page
		offset += 50

		userPosts, err = m.api.GetUserPosts(service, userId, offset)
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

	return m.processDownloadQueue(item, []api.QuickPost{{
		Service: postId[1],
		User:    postId[2],
		ID:      postId[3],
	}})
}
