package pawchive

import (
	"fmt"
	"regexp"

	"log/slog"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/internal/modules/pawchive/api"
)

func (m *pawchive) parseUser(item *models.TrackedItem) error {
	search := regexp.MustCompile(`https://pawchive\.st/([^/?&]+)/user/([^/?&]+)`).FindStringSubmatch(item.URI)
	service := ""
	userId := ""
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

	slog.Info(fmt.Sprintf("found user %s (%s)", profile.Name, profile.ID), "module", m.Key)

	userPosts, err := m.api.GetUserPosts(service, userId, 0)
	if err != nil {
		return fmt.Errorf("failed to fetch user posts: %w", err)
	}

	var (
		downloadQueue    []api.Post
		foundCurrentItem bool
		offset           int
	)

	for {
		for _, post := range userPosts {
			if post.ID == item.CurrentItem {
				foundCurrentItem = true
				break
			}
			downloadQueue = append(downloadQueue, post)
		}

		// stop on a short page (page size is 50)
		if foundCurrentItem || len(userPosts) < 50 {
			break
		}

		offset += 50
		userPosts, err = m.api.GetUserPosts(service, userId, offset)
		if err != nil {
			return fmt.Errorf("failed to fetch user posts: %w", err)
		}
	}

	// reverse the queue to download oldest items first
	for i, j := 0, len(downloadQueue)-1; i < j; i, j = i+1, j-1 {
		downloadQueue[i], downloadQueue[j] = downloadQueue[j], downloadQueue[i]
	}

	return m.processDownloadQueue(item, downloadQueue)
}

func (m *pawchive) parsePost(item *models.TrackedItem) error {
	// example: https://pawchive.st/patreon/user/4829343/post/161164023
	match := regexp.MustCompile(`.*/([^/?&]+)/user/([^/?&]+)/post/(\w+)`).FindStringSubmatch(item.URI)
	if len(match) != 4 {
		return fmt.Errorf("could not extract post ID from URL: %s", item.URI)
	}

	post, err := m.api.GetPostDetails(match[1], match[2], match[3])
	if err != nil {
		return fmt.Errorf("failed to fetch post details: %w", err)
	}

	return m.processDownloadQueue(item, []api.Post{*post})
}
