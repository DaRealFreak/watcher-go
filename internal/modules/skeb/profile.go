package skeb

import (
	"fmt"
	"log/slog"
	"path"
	"strconv"
	"strings"

	"github.com/DaRealFreak/watcher-go/internal/models"
	"github.com/DaRealFreak/watcher-go/pkg/fp"
)

type mediaItem struct {
	fileName string
	fileURI  string
}

func (m *skeb) parseProfile(item *models.TrackedItem) error {
	username := m.extractUsername(item.URI)

	slog.Info(
		fmt.Sprintf("parsing profile \"@%s\"", username),
		"module", m.Key,
	)

	currentNum := 0
	if item.CurrentItem != "" {
		currentNum, _ = strconv.Atoi(item.CurrentItem)
	}

	// collect new work numbers from the works list (API returns newest first)
	type pendingWork struct {
		username string
		postNum  string
		num      int
	}

	var newWorks []pendingWork
	offset := 0
	foundCurrentItem := false

	for {
		works, err := m.getWorksList(username, offset)
		if err != nil {
			return err
		}

		for _, work := range works {
			if work.Private {
				continue
			}

			postNum := extractPostNum(work.Path)
			num, _ := strconv.Atoi(postNum)

			if num <= currentNum {
				foundCurrentItem = true
				continue
			}

			workUser := extractUsernameFromPath(work.Path)
			newWorks = append(newWorks, pendingWork{
				username: workUser,
				postNum:  postNum,
				num:      num,
			})
		}

		if foundCurrentItem || len(works) < 30 {
			break
		}
		offset += 30
	}

	if len(newWorks) == 0 {
		return nil
	}

	// reverse to process oldest first
	for i, j := 0, len(newWorks)-1; i < j; i, j = i+1, j-1 {
		newWorks[i], newWorks[j] = newWorks[j], newWorks[i]
	}

	slog.Info(
		fmt.Sprintf("found %d new items for uri: \"%s\"", len(newWorks), item.URI),
		"module", m.Key,
	)

	// fetch each work individually to get previews, then download
	for i, pw := range newWorks {
		slog.Info(
			fmt.Sprintf(
				"downloading updates for uri: \"%s\" (%0.2f%%)",
				item.URI,
				float64(i+1)/float64(len(newWorks))*100,
			),
			"module", m.Key,
		)

		work, err := m.getWork(pw.username, pw.postNum)
		if err != nil {
			slog.Warn(
				fmt.Sprintf("failed to fetch work %s, skipping: %s", pw.postNum, err.Error()),
				"module", m.Key,
			)
			continue
		}

		if work.Private {
			continue
		}

		tag := username
		if work.Creator.ScreenName != "" {
			tag = work.Creator.ScreenName
		}

		items := m.extractMediaFromWork(*work)
		for _, mi := range items {
			filePath := path.Join(
				m.GetDownloadDirectory(),
				m.Key,
				fp.TruncateMaxLength(fp.SanitizePath(item.SubFolder, false)),
				fp.TruncateMaxLength(fp.SanitizePath(tag, false)),
				fp.TruncateMaxLength(fp.SanitizePath(mi.fileName, false)),
			)

			if err = m.Session.DownloadFile(filePath, mi.fileURI); err != nil {
				return err
			}
		}

		m.DbIO.UpdateTrackedItem(item, pw.postNum)
	}

	return nil
}

func (m *skeb) parseWork(item *models.TrackedItem) error {
	username, postNum := m.extractWorkParts(item.URI)

	work, err := m.getWork(username, postNum)
	if err != nil {
		return err
	}

	if work.Private {
		slog.Warn(
			fmt.Sprintf("work %s is private, skipping", postNum),
			"module", m.Key,
		)
		return nil
	}

	items := m.extractMediaFromWork(*work)
	if len(items) == 0 {
		return nil
	}

	tag := username
	if work.Creator.ScreenName != "" {
		tag = work.Creator.ScreenName
	}

	slog.Info(
		fmt.Sprintf("found %d new items for uri: \"%s\"", len(items), item.URI),
		"module", m.Key,
	)

	for _, mi := range items {
		filePath := path.Join(
			m.GetDownloadDirectory(),
			m.Key,
			fp.TruncateMaxLength(fp.SanitizePath(item.SubFolder, false)),
			fp.TruncateMaxLength(fp.SanitizePath(tag, false)),
			fp.TruncateMaxLength(fp.SanitizePath(mi.fileName, false)),
		)

		if err = m.Session.DownloadFile(filePath, mi.fileURI); err != nil {
			return err
		}
	}

	m.DbIO.UpdateTrackedItem(item, postNum)
	m.DbIO.ChangeTrackedItemCompleteStatus(item, true)
	return nil
}

func (m *skeb) extractUsername(uri string) string {
	uri = strings.TrimRight(uri, "/")
	parts := strings.Split(uri, "/")
	for _, part := range parts {
		if strings.HasPrefix(part, "@") {
			return strings.TrimPrefix(part, "@")
		}
	}
	return uri
}

func (m *skeb) extractWorkParts(uri string) (username, postNum string) {
	uri = strings.TrimRight(uri, "/")
	parts := strings.Split(uri, "/")
	for i, part := range parts {
		if strings.HasPrefix(part, "@") {
			username = strings.TrimPrefix(part, "@")
		}
		if part == "works" && i+1 < len(parts) {
			postNum = parts[i+1]
		}
	}
	return
}

// extractPostNum extracts the work number from a path like "/@username/works/9"
func extractPostNum(workPath string) string {
	parts := strings.Split(workPath, "/")
	return parts[len(parts)-1]
}

// extractUsernameFromPath extracts the username from a path like "/@username/works/9"
func extractUsernameFromPath(workPath string) string {
	parts := strings.Split(workPath, "/")
	for _, part := range parts {
		if strings.HasPrefix(part, "@") {
			return strings.TrimPrefix(part, "@")
		}
	}
	return ""
}

func (m *skeb) extractMediaFromWork(work workResponse) []mediaItem {
	var items []mediaItem

	for _, p := range work.Previews {
		ext := p.Information.Extension
		if ext == "" {
			ext = extractExtensionFromURL(p.URL)
		}

		items = append(items, mediaItem{
			fileName: fmt.Sprintf("%d_%d.%s", work.ID, p.ID, ext),
			fileURI:  p.URL,
		})
	}

	return items
}

func extractExtensionFromURL(rawURL string) string {
	// strip query parameters
	if idx := strings.Index(rawURL, "?"); idx >= 0 {
		rawURL = rawURL[:idx]
	}
	if idx := strings.LastIndex(rawURL, "."); idx >= 0 {
		return rawURL[idx+1:]
	}
	return "jpg"
}
